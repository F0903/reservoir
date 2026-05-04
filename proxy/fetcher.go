package proxy

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/headers"
	"reservoir/utils/countingreader"
	"reservoir/utils/syncmap"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

var ErrNotCacheable = errors.New("response not cacheable")

type fetchedBytesReadCloser struct {
	io.ReadCloser
}

func (r fetchedBytesReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		metrics.Global.Requests.BytesFetched.Add(int64(n))
	}
	return n, err
}

type fetcher struct {
	cache        cache.Cache[cachedRequestInfo]
	cfg          *config.Config
	policy       cachePolicy
	client       *http.Client
	group        singleflight.Group
	variantIndex *syncmap.SyncMap[cache.CacheKey, []string]
}

func newFetcher(cacheStore cache.Cache[cachedRequestInfo], cfg *config.Config, upstreamClient *http.Client) fetcher {
	if upstreamClient == nil {
		upstreamClient = newUpstreamClient()
	}

	return fetcher{
		cache:        cacheStore,
		cfg:          cfg,
		policy:       newCachePolicy(cfg),
		client:       upstreamClient,
		group:        singleflight.Group{},
		variantIndex: syncmap.New[cache.CacheKey, []string](),
	}
}

func normalizeVaryHeaderValues(values []string) string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		parts := make([]string, 0)
		for part := range strings.SplitSeq(value, ",") {
			part = strings.ToLower(strings.TrimSpace(part))
			if part == "" {
				continue
			}
			parts = append(parts, part)
		}
		normalized = append(normalized, strings.Join(parts, ","))
	}
	return strings.Join(normalized, ",")
}

func makeVariantCacheKey(req *http.Request, baseKey cache.CacheKey, vary []string) cache.CacheKey {
	if len(vary) == 0 {
		return baseKey
	}

	parts := []string{baseKey.Hex}
	for _, headerName := range vary {
		values := req.Header.Values(http.CanonicalHeaderKey(headerName))
		parts = append(parts, headerName+"="+normalizeVaryHeaderValues(values))
	}
	return cache.FromString(strings.Join(parts, "|"))
}

func (f *fetcher) lookupCacheKey(req *http.Request, baseKey cache.CacheKey) cache.CacheKey {
	vary, _ := f.variantIndex.Get(baseKey)
	return makeVariantCacheKey(req, baseKey, vary)
}

func (f *fetcher) singleflightKey(req *http.Request, baseKey cache.CacheKey) string {
	return makeVariantCacheKey(req, baseKey, supportedVaryHeaders).Hex
}

func (f *fetcher) setVariantIndex(baseKey cache.CacheKey, vary []string) {
	if len(vary) == 0 {
		f.variantIndex.Delete(baseKey)
		return
	}

	copied := make([]string, len(vary))
	copy(copied, vary)
	f.variantIndex.Set(baseKey, copied)
}

func (f *fetcher) closeIdleConnections() {
	f.client.CloseIdleConnections()
}

func trackFetchedBytes(body io.ReadCloser) io.ReadCloser {
	return fetchedBytesReadCloser{ReadCloser: body}
}

func (f *fetcher) handleUpstream304(req *http.Request, key cache.CacheKey) (cached *cache.Entry[cachedRequestInfo], err error) {
	slog.Debug("Handling 304 response from upstream", "url", req.URL, "key", key)

	slog.Debug("Revalidating cache metadata...", "url", req.URL, "key", key)
	err = f.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[cachedRequestInfo]) {
		// Update the metadata to reflect that the cached response is still valid.
		maxAge := f.cfg.Proxy.CachePolicy.DefaultMaxAge.Read().Cast()
		meta.Expires = time.Now().Add(maxAge)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpdateCacheMetadata, err)
	}

	slog.Debug("Successfully revalidated cache metadata", "url", req.URL, "key", key)
	return f.cache.Get(key)
}

func (f *fetcher) handleUpstream200(req *http.Request, resp *http.Response, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives, upstreamHd *headers.HeaderDirectives) (cached *cache.Entry[cachedRequestInfo], err error) {
	slog.Debug("Handling 200 response from upstream", "url", req.URL, "key", lookupKey)

	decision := f.policy.Decide(req, resp, upstreamHd)
	if !decision.Cacheable {
		slog.Debug("Got response code 200, but result is not cacheable", "status", resp.Status, "url", req.URL, "key", lookupKey, "reason", decision.Reason)
		return nil, nil
	}

	storeKey := makeVariantCacheKey(req, baseKey, decision.Vary)

	slog.Debug("Caching response...", "status", resp.Status, "url", req.URL, "key", storeKey, "lookup_key", lookupKey)

	lastModified := time.Now()
	if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
		lastModified = t
	}

	etag := resp.Header.Get("ETag")

	var bytesRead int
	reader := countingreader.New(resp.Body, &bytesRead)
	cacheReader := cache.WithSizeHint(reader, resp.ContentLength)

	cached, err = f.cache.Cache(storeKey, cacheReader, decision.Expires, cachedRequestInfo{
		ETag:         etag,
		LastModified: lastModified,
		Header:       resp.Header,
		Vary:         decision.Vary,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCacheResponseFailed, err)
	}
	f.setVariantIndex(baseKey, decision.Vary)

	metrics.Global.Requests.BytesFetched.Add(int64(bytesRead))
	slog.Info("Successfully cached response", "status", resp.Status, "url", req.URL, "key", storeKey, "expires", decision.Expires)
	return cached, nil
}

func (f *fetcher) handleUpstream416(req *http.Request, resp *http.Response, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives, noRetry bool) (cached *cache.Entry[cachedRequestInfo], err error) {
	slog.Debug("Upstream responded with 416 Range Not Satisfiable, retrying without Range header...", "url", req.URL)

	if noRetry || !f.cfg.Proxy.RetryOnRange416.Read() {
		slog.Debug("Not retrying 416 Range Not Satisfiable. Returning as is.", "url", req.URL)
		return nil, nil
	}

	// Close the previous response body to avoid resource leaks
	resp.Body.Close()

	retryReq := req.Clone(req.Context())
	clientHd.Range.SyncRemove(retryReq.Header)

	retryResp, _, err := f.sendRequestToUpstream(retryReq)
	if err != nil {
		slog.Error("Error fetching upstream on retry without Range header", "url", req.URL, "error", err)
		return nil, err
	}
	*resp = *retryResp // Replace the original response with the new one

	return f.handleUpstreamResponse(retryReq, resp, baseKey, lookupKey, clientHd, true)
}

func (f *fetcher) handleUpstreamResponse(req *http.Request, resp *http.Response, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives, noRetry bool) (cached *cache.Entry[cachedRequestInfo], err error) {
	slog.Debug("Preparing to handle upstream response...", "url", req.URL, "status", resp.StatusCode)

	upstreamHd := headers.ParseHeaderDirective(resp.Header)

	switch resp.StatusCode {
	case http.StatusOK:
		return f.handleUpstream200(req, resp, baseKey, lookupKey, clientHd, upstreamHd)
	case http.StatusNotModified:
		return f.handleUpstream304(req, lookupKey)
	case http.StatusRequestedRangeNotSatisfiable:
		return f.handleUpstream416(req, resp, baseKey, lookupKey, clientHd, noRetry)
	default:
		slog.Debug("Upstream returned non-cachable response", "url", req.URL, "status", resp.StatusCode)
		return nil, nil
	}
}

func (f *fetcher) sendRequestToUpstream(req *http.Request) (*http.Response, time.Duration, error) {
	slog.Debug("Sending request to upstream", "url", req.URL)
	metrics.Global.Requests.UpstreamRequests.Increment()

	startTime := time.Now()
	resp, err := sendRequestToTarget(f.client, req, f.cfg.Proxy.UpstreamDefaultHttps.Read())
	latency := time.Since(startTime)

	metrics.Global.Requests.UpstreamRequestLatency.Add(latency.Nanoseconds())
	if err != nil {
		return nil, 0, err
	}

	slog.Debug("Received response from upstream", "url", req.URL, "status", resp.Status, "latency_ns", latency.Nanoseconds())
	return resp, latency, nil
}

func (f *fetcher) fetchUpstream(req *http.Request, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives) (fetchResult, error) {
	slog.Debug("Fetching from upstream...")

	resp, upstreamLatency, err := f.sendRequestToUpstream(req)
	if err != nil {
		slog.Error("Error fetching upstream", "url", req.URL, "error", err)
		return fetchResult{}, err
	}

	cached, err := f.handleUpstreamResponse(req, resp, baseKey, lookupKey, clientHd, false)
	if err != nil {
		resp.Body.Close()
		slog.Error("Error handling upstream response after cache miss", "url", req.URL, "error", err)
		return fetchResult{}, err
	}

	if cached == nil {
		slog.Debug("Upstream response is not cachable, returning direct result after cache miss...", "url", req.URL, "status", resp.StatusCode)

		resp.Body = trackFetchedBytes(resp.Body)

		fetchInfo := fetchInfo{UpstreamStatus: resp.StatusCode, Status: hitStatusMiss, UpstreamLatency: upstreamLatency}
		directRes := directFetchResult{Response: resp, fetchInfo: fetchInfo}
		return fetchResult{Type: fetchTypeDirect, Direct: directRes}, nil
	}

	// We have a cached entry, so we don't need the original upstream response body anymore.
	resp.Body.Close()

	slog.Debug("Returning cached fetch result...")

	fetchInfo := fetchInfo{UpstreamStatus: resp.StatusCode, Status: hitStatusMiss, UpstreamLatency: upstreamLatency}
	cachedResult := cachedFetchResult{fetchInfo: fetchInfo, Entry: cached}
	return fetchResult{Type: fetchTypeCached, Cached: cachedResult}, nil
}

// Fetches directly from upstream without caching.
// Used for requests where you don't care about caching, and just want to pass it straight back to the client.
func (f *fetcher) fetchDirectlyFromUpstream(req *http.Request) (fetchResult, error) {
	slog.Debug("Fetching directly from upstream...")
	resp, upstreamLatency, err := f.sendRequestToUpstream(req)
	if err != nil {
		slog.Error("Error fetching upstream", "url", req.URL, "error", err)
		return fetchResult{}, err
	}

	resp.Body = trackFetchedBytes(resp.Body)

	slog.Debug("Returning direct fetch result", "url", req.URL, "status", resp.StatusCode)
	fetchInfo := fetchInfo{UpstreamStatus: resp.StatusCode, Status: hitStatusMiss, UpstreamLatency: upstreamLatency}
	directRes := directFetchResult{Response: resp, fetchInfo: fetchInfo}
	return fetchResult{Type: fetchTypeDirect, Direct: directRes}, nil
}

func (f *fetcher) handleCacheMiss(req *http.Request, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives) (fetchResult, error) {
	slog.Debug("Cache miss, fetching upstream...", "url", req.URL, "key", lookupKey)
	upFetch, err := f.fetchUpstream(req, baseKey, lookupKey, clientHd)
	if err != nil {
		return fetchResult{}, err
	}

	return upFetch, nil
}

func (f *fetcher) serveStaleCachedResponse(req *http.Request, lookupKey cache.CacheKey, upstreamStatus int, upstreamErr error) (fetchResult, error) {
	if upstreamErr != nil {
		slog.Warn("Serving stale cached response because upstream revalidation failed", "url", req.URL, "key", lookupKey, "error", upstreamErr)
	} else {
		slog.Warn("Serving stale cached response because upstream returned an error status", "url", req.URL, "key", lookupKey, "upstream_status", upstreamStatus)
	}

	cached, err := f.cache.Get(lookupKey)
	if err != nil {
		if upstreamErr != nil {
			return fetchResult{}, upstreamErr
		}
		return fetchResult{}, err
	}

	return fetchResult{
		Type: fetchTypeCached,
		Cached: cachedFetchResult{
			fetchInfo: fetchInfo{
				UpstreamStatus: upstreamStatus,
				Status:         hitStatusStale,
			},
			Entry: cached,
		},
	}, nil
}

// Fetches the requested resource either from cache or upstream.
func (f *fetcher) getFromCacheOrFetch(req *http.Request, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives) (fetchResult, error) {
	slog.Debug("Trying to get request from cache...")

	cached, err := f.cache.Get(lookupKey)
	if err != nil {
		if errors.Is(err, cache.ErrCacheEntryNotFound) {
			slog.Debug("Cache miss, will fetch from upstream.", "url", req.URL, "key", lookupKey)
			res, err := f.handleCacheMiss(req, baseKey, lookupKey, clientHd)
			if err != nil {
				return fetchResult{}, err
			}
			if res.Type == fetchTypeDirect {
				res.Direct.Response.Body.Close()
				return fetchResult{}, ErrNotCacheable
			}
			return res, nil
		}

		metrics.Global.Cache.CacheErrors.Increment()
		// We just return ErrNotCacheable to trigger direct fetch in caller
		return fetchResult{}, ErrNotCacheable
	}

	if !cached.Stale {
		slog.Debug("Cache hit, returning cached response.", "url", req.URL, "key", lookupKey)
		fetchInfo := fetchInfo{Status: hitStatusHit}

		cachedResult := cachedFetchResult{fetchInfo: fetchInfo, Entry: cached}
		return fetchResult{Type: fetchTypeCached, Cached: cachedResult}, nil
	}

	slog.Debug("Cached response is stale, fetching upstream.", "url", req.URL, "key", lookupKey)

	// Close the stale file handle before fetching from upstream
	if cached.Data != nil {
		cached.Data.Close()
		cached.Data = nil
	}

	up := req.Clone(req.Context())

	// Cache is stale: set conditional headers if available
	if cached.Metadata.Object.ETag != "" {
		up.Header.Set("If-None-Match", cached.Metadata.Object.ETag)
	}
	if !cached.Metadata.Object.LastModified.IsZero() {
		up.Header.Set("If-Modified-Since", cached.Metadata.Object.LastModified.Format(http.TimeFormat))
	}

	fetch, err := f.fetchUpstream(up, baseKey, lookupKey, clientHd)
	if err != nil {
		return f.serveStaleCachedResponse(req, lookupKey, 0, err)
	}
	if fetch.Type == fetchTypeDirect {
		if fetch.Direct.UpstreamStatus >= 500 {
			status := fetch.Direct.UpstreamStatus
			fetch.Direct.Response.Body.Close()
			return f.serveStaleCachedResponse(req, lookupKey, status, nil)
		}

		// The upstream response is authoritative but not cacheable. Return it
		// directly instead of retrying a second upstream request.
		slog.Debug("Stale revalidation returned a non-cacheable direct response", "url", req.URL, "status", fetch.Direct.UpstreamStatus)
		return fetch, nil
	}
	fetch.getFetchInfoRef().Status = hitStatusRevalidated

	return fetch, nil
}

// Will deduplicate cachable requests and otherwise return the bypassed upstream response.
// IMPORTANT: Remember to close data streams!
func (f *fetcher) dedupFetch(req *http.Request, baseKey cache.CacheKey, clientHd *headers.HeaderDirectives) (fetched fetchResult, err error) {
	slog.Debug("Attempting to dedup fetch...")
	lookupKey := f.lookupCacheKey(req, baseKey)

	if !f.policy.RequestAllowsSharedCache(req) {
		slog.Debug("Request contains credentials, bypassing shared cache", "url", req.URL)
		metrics.Global.Requests.NonCoalescedRequests.Increment()
		return f.fetchDirectlyFromUpstream(req)
	}

	shouldCoalesce := !clientHd.Range.IsPresent() && req.Method == http.MethodGet
	if !shouldCoalesce {
		// These requests also aren't cacheable, so they just go straight to upstream..
		slog.Debug("Request can't be coalesced, fetching upstream...")
		metrics.Global.Requests.NonCoalescedRequests.Increment()

		return f.fetchUpstream(req, baseKey, lookupKey, clientHd)
	}

	originalClientHd := *clientHd // Copy the original client headers so the shared requests don't get a modified version

	fetchedObj, err, shared := f.group.Do(f.singleflightKey(req, baseKey), func() (any, error) {
		return f.getFromCacheOrFetch(req, baseKey, lookupKey, clientHd)
	})
	if err != nil {
		if errors.Is(err, ErrNotCacheable) {
			slog.Debug("Request was not cacheable in singleflight, falling back to direct fetch", "url", req.URL)
			metrics.Global.Requests.NonCoalescedRequests.Increment()
			return f.fetchDirectlyFromUpstream(req)
		}
		return fetchResult{}, err
	}
	fetched = fetchedObj.(fetchResult)

	// If shared is true, this means the request was coalesced.
	// Meaning that the result is being shared with other requests in-flight.

	if shared {
		metrics.Global.Requests.CoalescedRequests.Increment()

		// Restore the original client headers for shared requests
		clientHd = &originalClientHd
	} else {
		metrics.Global.Requests.NonCoalescedRequests.Increment()
	}

	switch fetched.Type {
	case fetchTypeCached:
		slog.Debug("Fetched cached response", "url", req.URL, "status", fetched.Cached.Status, "upstream_status", fetched.Cached.UpstreamStatus, "coalesced", shared)
		fetched.Cached.Coalesced = shared

		// If the result is shared, we cannot share the same Data (seeker) between goroutines.
		// We close the shared handle and each caller gets their own fresh one.
		if shared {
			if fetched.Cached.Entry.Data != nil {
				fetched.Cached.Entry.Data.Close()
			}

			freshLookupKey := f.lookupCacheKey(req, baseKey)
			cached, err := f.cache.Get(freshLookupKey)
			if err != nil {
				slog.Error("Error getting newly cached response. Bypassing cache and fetching upstream...", "url", req.URL, "key", freshLookupKey, "error", err)
				return f.fetchDirectlyFromUpstream(req)
			}
			fetched.Cached.Entry = cached
		}

		// Track coalesced cache hits/misses
		if shared {
			switch fetched.Cached.Status {
			case hitStatusHit:
				metrics.Global.Requests.CoalescedCacheHits.Increment()
			case hitStatusRevalidated:
				metrics.Global.Requests.CoalescedCacheRevalidations.Increment()
			case hitStatusMiss:
				metrics.Global.Requests.CoalescedCacheMisses.Increment()
			}
		}

		slog.Debug("Fetched cached response", "url", req.URL, "status", fetched.Cached.Status, "upstream_status", fetched.Cached.UpstreamStatus, "coalesced", shared)
		return fetched, nil

	case fetchTypeDirect:
		if shared {
			slog.Debug("Fetched shared direct response, fetching own upstream...", "url", req.URL, "status", fetched.Direct.Status, "upstream_status", fetched.Direct.UpstreamStatus, "coalesced", shared)
			return f.fetchDirectlyFromUpstream(req) // Followers should fetch their own upstream response
		}
		slog.Debug("Fetched direct response", "url", req.URL, "status", fetched.Direct.Status, "upstream_status", fetched.Direct.UpstreamStatus, "coalesced", shared)
		return fetched, nil

	default:
		// This should not be possible unless new fetch types are introduced and not correctly implemented.
		slog.Error("Unknown fetch result type", "type", fetched.Type)
		return fetchResult{}, errors.ErrUnsupported
	}
}
