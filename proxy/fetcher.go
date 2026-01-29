package proxy

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/headers"
	"reservoir/utils/countingreader"
	"time"

	"golang.org/x/sync/singleflight"
)

var ErrNotCacheable = errors.New("response not cacheable")

type fetcher struct {
	cache cache.Cache[cachedRequestInfo]
	group singleflight.Group
}

func newFetcher(cache cache.Cache[cachedRequestInfo]) fetcher {
	return fetcher{
		cache: cache,
		group: singleflight.Group{},
	}
}

func shouldResponseBeCached(resp *http.Response, upstreamHd *headers.HeaderDirectives) bool {
	return upstreamHd.ShouldCache() &&
		resp.StatusCode == http.StatusOK &&
		resp.Request.Method == http.MethodGet
}

func (f *fetcher) handleUpstream304(req *http.Request, key cache.CacheKey) (cached *cache.Entry[cachedRequestInfo], err error) {
	slog.Debug("Handling 304 response from upstream", "url", req.URL, "key", key)

	slog.Debug("Revalidating cache metadata...", "url", req.URL, "key", key)
	err = f.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[cachedRequestInfo]) {
		// Update the metadata to reflect that the cached response is still valid.
		maxAge := config.Global.DefaultCacheMaxAge.Read().Cast()
		meta.Expires = time.Now().Add(maxAge)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpdateCacheMetadata, err)
	}

	slog.Debug("Successfully revalidated cache metadata", "url", req.URL, "key", key)
	return f.cache.Get(key)
}

func (f *fetcher) handleUpstream200(req *http.Request, resp *http.Response, key cache.CacheKey, upstreamHd *headers.HeaderDirectives) (cached *cache.Entry[cachedRequestInfo], err error) {
	slog.Debug("Handling 200 response from upstream", "url", req.URL, "key", key)

	if !shouldResponseBeCached(resp, upstreamHd) {
		slog.Debug("Got response code 200, but result is not cacheable", "status", resp.Status, "url", req.URL, "key", key)
		return nil, nil
	}

	slog.Debug("Caching response...", "status", resp.Status, "url", req.URL, "key", key)

	lastModified := time.Now()
	if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
		lastModified = t
	}

	etag := resp.Header.Get("ETag")

	var bytesRead int
	reader := countingreader.New(resp.Body, &bytesRead)

	maxAge := upstreamHd.GetExpiresOrDefault()
	cached, err = f.cache.Cache(key, reader, maxAge, cachedRequestInfo{
		ETag:         etag,
		LastModified: lastModified,
		Header:       resp.Header,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCacheResponseFailed, err)
	}

	metrics.Global.Requests.BytesFetched.Add(int64(bytesRead))
	slog.Info("Successfully cached response", "status", resp.Status, "url", req.URL, "key", key, "expires_in", maxAge)
	return cached, nil
}

func (f *fetcher) handleUpstream416(req *http.Request, resp *http.Response, key cache.CacheKey, clientHd *headers.HeaderDirectives, noRetry bool) (cached *cache.Entry[cachedRequestInfo], err error) {
	slog.Debug("Upstream responded with 416 Range Not Satisfiable, retrying without Range header...", "url", req.URL)

	if noRetry {
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

	return f.handleUpstreamResponse(retryReq, resp, key, clientHd, true)
}

func (f *fetcher) handleUpstreamResponse(req *http.Request, resp *http.Response, key cache.CacheKey, clientHd *headers.HeaderDirectives, noRetry bool) (cached *cache.Entry[cachedRequestInfo], err error) {
	slog.Debug("Preparing to handle upstream response...", "url", req.URL, "status", resp.StatusCode)

	upstreamHd := headers.ParseHeaderDirective(resp.Header)

	switch resp.StatusCode {
	case http.StatusOK:
		return f.handleUpstream200(req, resp, key, upstreamHd)
	case http.StatusNotModified:
		return f.handleUpstream304(req, key)
	case http.StatusRequestedRangeNotSatisfiable:
		return f.handleUpstream416(req, resp, key, clientHd, noRetry)
	default:
		slog.Debug("Upstream returned non-cachable response", "url", req.URL, "status", resp.StatusCode)
		return nil, nil
	}
}

func (f *fetcher) sendRequestToUpstream(req *http.Request) (*http.Response, time.Duration, error) {
	slog.Debug("Sending request to upstream", "url", req.URL)
	metrics.Global.Requests.UpstreamRequests.Increment()

	startTime := time.Now()
	resp, err := sendRequestToTarget(req, config.Global.UpstreamDefaultHttps.Read())
	latency := time.Since(startTime)

	metrics.Global.Requests.UpstreamRequestLatency.Add(latency.Nanoseconds())
	if err != nil {
		return nil, 0, err
	}

	slog.Debug("Received response from upstream", "url", req.URL, "status", resp.Status, "latency_ns", latency.Nanoseconds())
	return resp, latency, nil
}

func (f *fetcher) fetchUpstream(req *http.Request, key cache.CacheKey, clientHd *headers.HeaderDirectives) (fetchResult, error) {
	slog.Debug("Fetching from upstream...")

	resp, upstreamLatency, err := f.sendRequestToUpstream(req)
	if err != nil {
		slog.Error("Error fetching upstream", "url", req.URL, "error", err)
		return fetchResult{}, err
	}

	cached, err := f.handleUpstreamResponse(req, resp, key, clientHd, false)
	if err != nil {
		resp.Body.Close()
		slog.Error("Error handling upstream response after cache miss", "url", req.URL, "error", err)
		return fetchResult{}, err
	}

	if cached == nil {
		slog.Debug("Upstream response is not cachable, returning direct result after cache miss...", "url", req.URL, "status", resp.StatusCode)

		var bytesRead int
		countingBody := countingreader.NewReadCloser(resp.Body, &bytesRead)
		resp.Body = countingBody

		metrics.Global.Requests.BytesFetched.Add(int64(bytesRead))

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

	var bytesRead int
	countingBody := countingreader.NewReadCloser(resp.Body, &bytesRead)
	resp.Body = countingBody

	metrics.Global.Requests.BytesFetched.Add(int64(bytesRead))

	slog.Debug("Returning direct fetch result", "url", req.URL, "status", resp.StatusCode)
	fetchInfo := fetchInfo{UpstreamStatus: resp.StatusCode, Status: hitStatusMiss, UpstreamLatency: upstreamLatency}
	directRes := directFetchResult{Response: resp, fetchInfo: fetchInfo}
	return fetchResult{Type: fetchTypeDirect, Direct: directRes}, nil
}

func (f *fetcher) handleCacheMiss(req *http.Request, key cache.CacheKey, clientHd *headers.HeaderDirectives) (fetchResult, error) {
	slog.Debug("Cache miss, fetching upstream...", "url", req.URL, "key", key)
	upFetch, err := f.fetchUpstream(req, key, clientHd)
	if err != nil {
		return fetchResult{}, err
	}

	return upFetch, nil
}

// Fetches the requested resource either from cache or upstream.
func (f *fetcher) getFromCacheOrFetch(req *http.Request, key cache.CacheKey, clientHd *headers.HeaderDirectives) (fetchResult, error) {
	slog.Debug("Trying to get request from cache...")

	cached, err := f.cache.Get(key)
	if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			slog.Debug("Cache miss, will fetch from upstream.", "url", req.URL, "key", key)
			res, err := f.handleCacheMiss(req, key, clientHd)
			if err != nil {
				return fetchResult{}, err
			}
			if res.Type == fetchTypeDirect {
				res.Direct.Response.Body.Close()
				return fetchResult{}, ErrNotCacheable
			}
			if res.Type == fetchTypeCached && res.Cached.Entry.Data != nil {
				res.Cached.Entry.Data.Close()
				res.Cached.Entry.Data = nil
			}
			return res, nil
		}

		metrics.Global.Cache.CacheErrors.Increment()
		// We just return ErrNotCacheable to trigger direct fetch in caller
		return fetchResult{}, ErrNotCacheable
	}

	if !cached.Stale {
		slog.Debug("Cache hit, returning cached response.", "url", req.URL, "key", key)
		fetchInfo := fetchInfo{Status: hitStatusHit}

		// Close the file handle to ensure we don't pass open files through singleflight
		if cached.Data != nil {
			cached.Data.Close()
			cached.Data = nil
		}

		cachedResult := cachedFetchResult{fetchInfo: fetchInfo, Entry: cached}
		return fetchResult{Type: fetchTypeCached, Cached: cachedResult}, nil
	}

	slog.Debug("Cached response is stale, fetching upstream.", "url", req.URL, "key", key)

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

	fetch, err := f.fetchUpstream(up, key, clientHd)
	if err != nil {
		return fetchResult{}, err
	}
	if fetch.Type == fetchTypeDirect {
		fetch.Direct.Response.Body.Close()
		return fetchResult{}, ErrNotCacheable
	}
	if fetch.Type == fetchTypeCached && fetch.Cached.Entry.Data != nil {
		fetch.Cached.Entry.Data.Close()
		fetch.Cached.Entry.Data = nil
	}

	fetch.getFetchInfoRef().Status = hitStatusRevalidated

	return fetch, nil
}

// Will deduplicate cachable requests and otherwise return the bypassed upstream response.
// IMPORTANT: Remember to close data streams!
func (f *fetcher) dedupFetch(req *http.Request, key cache.CacheKey, clientHd *headers.HeaderDirectives) (fetched fetchResult, err error) {
	slog.Debug("Attempting to dedup fetch...")

	shouldCoalesce := !clientHd.Range.IsPresent() && req.Method == http.MethodGet
	if !shouldCoalesce {
		// These requests also aren't cacheable, so they just go straight to upstream..
		slog.Debug("Request can't be coalesced, fetching upstream...")
		metrics.Global.Requests.NonCoalescedRequests.Increment()

		return f.fetchUpstream(req, key, clientHd)
	}

	originalClientHd := *clientHd // Copy the original client headers so the shared requests don't get a modified version

	fetchedObj, err, shared := f.group.Do(key.Hex, func() (any, error) {
		return f.getFromCacheOrFetch(req, key, clientHd)
	})
	if err != nil {
		if errors.Is(err, ErrNotCacheable) {
			slog.Debug("Request was not cacheable in singleflight, falling back to direct fetch", "url", req.URL)
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
	}

	switch fetched.Type {
	case fetchTypeCached:
		slog.Debug("Fetched cached response", "url", req.URL, "status", fetched.Cached.Status, "upstream_status", fetched.Cached.UpstreamStatus, "coalesced", shared)
		fetched.Cached.Coalesced = shared

		// The Entry coming from singleflight has a nil Data field (closed).
		// We must open a fresh file handle for this request.
		cached, err := f.cache.Get(key)
		if err != nil {
			slog.Error("Error getting newly cached response. Bypassing cache and fetching upstream...", "url", req.URL, "key", key, "error", err)
			return f.fetchDirectlyFromUpstream(req)
		}
		fetched.Cached.Entry = cached

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
