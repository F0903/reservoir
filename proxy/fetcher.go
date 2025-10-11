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
	"time"

	"golang.org/x/sync/singleflight"
)

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

func (f *fetcher) handleUpstream304(req *http.Request, key cache.CacheKey) error {
	slog.Info("Handling 304 response from upstream", "url", req.URL, "key", key)

	var defaultCacheMaxAge time.Duration
	cfgLock := config.Global.Immutable()
	cfgLock.Read(func(c *config.Config) {
		defaultCacheMaxAge = c.DefaultCacheMaxAge.Read().Cast()
	})

	slog.Info("Revalidating cache metadata...", "url", req.URL, "key", key)
	err := f.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[cachedRequestInfo]) {
		// Update the metadata to reflect that the cached response is still valid.
		maxAge := defaultCacheMaxAge
		meta.Expires = time.Now().Add(maxAge)
	})
	if err != nil {
		slog.Error("Error updating cache metadata", "url", req.URL, "key", key, "error", err)
		return fmt.Errorf("%w: %v", ErrUpdateCacheMetadata, err)
	}

	return nil
}

func (f *fetcher) handleUpstream200(req *http.Request, resp *http.Response, key cache.CacheKey, upstreamHd *headers.HeaderDirectives) error {
	slog.Info("Handling 200 response from upstream", "url", req.URL, "key", key)

	if !shouldResponseBeCached(resp, upstreamHd) {
		slog.Info("Response is not cachable, returning direct...", "status", resp.Status, "url", req.URL, "key", key)
		return nil
	}

	slog.Info("Caching response...", "status", resp.Status, "url", req.URL, "key", key)

	lastModified := time.Now()
	if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
		lastModified = t
	}

	etag := resp.Header.Get("ETag")

	maxAge := upstreamHd.GetExpiresOrDefault()
	err := f.cache.Cache(key, resp.Body, maxAge, cachedRequestInfo{
		ETag:         etag,
		LastModified: lastModified,
		Header:       resp.Header,
	})
	if err != nil {
		slog.Error("Error caching response", "url", req.URL, "key", key, "error", err)
		return fmt.Errorf("%w: %v", ErrCacheResponseFailed, err)
	}

	return nil
}

func (f *fetcher) handleUpstreamResponse(req *http.Request, resp *http.Response, key cache.CacheKey) (cachable bool, err error) {
	slog.Debug("Preparing to handle upstream response...", "url", req.URL, "status", resp.StatusCode)

	upstreamHd := headers.ParseHeaderDirective(resp.Header)

	switch resp.StatusCode {
	case http.StatusOK:
		return true, f.handleUpstream200(req, resp, key, upstreamHd)
	case http.StatusNotModified:
		return true, f.handleUpstream304(req, key)
	default:
		slog.Debug("Upstream returned non-cachable response", "url", req.URL, "status", resp.StatusCode)
		return false, nil
	}
}

func (f *fetcher) fetchUpstream(req *http.Request) (fetchResult, error) {
	var upstreamDefaultHttps bool
	cfgLock := config.Global.Immutable()
	cfgLock.Read(func(c *config.Config) {
		upstreamDefaultHttps = c.UpstreamDefaultHttps.Read()
	})

	slog.Info("Sending request to upstream", "url", req.URL)
	resp, err := sendRequestToTarget(req, upstreamDefaultHttps)
	if err != nil {
		slog.Error("Error sending request to upstream target", "url", req.URL, "error", err)
		return fetchResult{}, err
	}
	slog.Info("Received response from upstream", "url", req.URL, "status", resp.Status)

	fetchInfo := fetchInfo{UpstreamStatus: resp.StatusCode, Status: hitStatusMiss}
	directRes := directFetchResult{Response: resp, fetchInfo: fetchInfo}
	return fetchResult{Type: fetchTypeDirect, Direct: directRes}, nil
}

func (f *fetcher) handleCacheMiss(req *http.Request, key cache.CacheKey) (fetchResult, error) {
	slog.Info("Cache miss, fetching upstream...", "url", req.URL, "key", key)
	upFetch, err := f.fetchUpstream(req)
	if err != nil {
		slog.Error("Error fetching upstream after cache miss", "url", req.URL, "error", err)
		return fetchResult{}, err
	}

	cachable, err := f.handleUpstreamResponse(req, upFetch.Direct.Response, key)
	if err != nil {
		slog.Error("Error handling upstream response after cache miss", "url", req.URL, "error", err)
		upFetch.Direct.Response.Body.Close()
		return fetchResult{}, err
	}

	if !cachable {
		slog.Info("Upstream response is not cachable, returning direct result after cache miss", "url", req.URL, "status", upFetch.Direct.Response.StatusCode)
		fetchInfo := fetchInfo{UpstreamStatus: upFetch.Direct.Response.StatusCode, Status: hitStatusMiss}
		directRes := directFetchResult{Response: upFetch.Direct.Response, fetchInfo: fetchInfo}
		return fetchResult{Type: fetchTypeDirect, Direct: directRes}, nil
	}
	upFetch.Direct.Response.Body.Close()

	slog.Info("Fetched and cached response after cache miss", "url", req.URL, "status", upFetch.Direct.Response.StatusCode, "upstream_status", upFetch.Direct.Response.StatusCode, "coalesced", false)
	fetchInfo := fetchInfo{Status: hitStatusMiss, UpstreamStatus: upFetch.Direct.Response.StatusCode}
	cachedResult := cachedFetchResult{fetchInfo: fetchInfo}
	return fetchResult{Type: fetchTypeCached, Cached: cachedResult}, nil
}

// Fetches the requested resource either from cache or upstream.
func (f *fetcher) internalDedupFetch(req *http.Request, key cache.CacheKey) (fetchResult, error) {
	slog.Info("Trying to get request from cache...")

	// Don't Get here since we are in the group lock.
	cachedMeta, stale, err := f.cache.GetMetadata(key)
	if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			slog.Info("Cache miss, will fetch from upstream.", "url", req.URL, "key", key)
			return f.handleCacheMiss(req, key)
		}

		slog.Error("Error getting cache for key", "key", key, "error", err)
		metrics.Global.Cache.CacheErrors.Increment()
		// We just log the error and fetch upstream
		return f.fetchUpstream(req)
	}

	if !stale {
		slog.Info("Cache hit, returning cached response.", "url", req.URL, "key", key)
		fetchInfo := fetchInfo{Status: hitStatusHit}
		cachedResult := cachedFetchResult{fetchInfo: fetchInfo}
		return fetchResult{Type: fetchTypeCached, Cached: cachedResult}, nil
	}

	slog.Info("Cached response is stale, fetching upstream.", "url", req.URL, "key", key)

	up := req.Clone(req.Context())
	fetchStatus := hitStatusRevalidated

	// Cache is stale: set conditional headers if available
	if cachedMeta.Object.ETag != "" {
		up.Header.Set("If-None-Match", cachedMeta.Object.ETag)
	}
	if !cachedMeta.Object.LastModified.IsZero() {
		up.Header.Set("If-Modified-Since", cachedMeta.Object.LastModified.Format(http.TimeFormat))
	}

	upFetch, err := f.fetchUpstream(up)
	if err != nil {
		slog.Error("Error fetching upstream", "url", up.URL, "error", err)
		return fetchResult{}, err
	}

	resp := upFetch.Direct.Response

	cachable, err := f.handleUpstreamResponse(up, resp, key)
	if err != nil {
		resp.Body.Close()
		slog.Error("Error handling upstream response", "url", up.URL, "error", err)
		return fetchResult{}, err
	}

	if !cachable {
		slog.Info("Upstream response is not cachable, returning direct result", "url", up.URL, "status", resp.StatusCode)
		fetchInfo := fetchInfo{Status: fetchStatus, UpstreamStatus: resp.StatusCode}
		directRes := directFetchResult{Response: resp, fetchInfo: fetchInfo}
		return fetchResult{Type: fetchTypeDirect, Direct: directRes}, nil
	}
	resp.Body.Close()

	fetchInfo := fetchInfo{Status: fetchStatus, UpstreamStatus: resp.StatusCode}
	cachedResult := cachedFetchResult{fetchInfo: fetchInfo}
	return fetchResult{Type: fetchTypeCached, Cached: cachedResult}, nil
}

// Will deduplicate cachable requests and otherwise return the bypassed upstream response.
// IMPORTANT: Remember to close data streams!
func (f *fetcher) dedupFetch(req *http.Request, key cache.CacheKey, clientHd *headers.HeaderDirectives) (fetched fetchResult, err error) {
	slog.Debug("Attempting to dedup fetch...")

	shouldCoalesce := clientHd.Range.IsNone() && req.Method == http.MethodGet
	if !shouldCoalesce {
		slog.Debug("Request can't be coalesced, fetching upstream...")
		metrics.Global.Requests.NonCoalescedRequests.Increment()
		return f.fetchUpstream(req)
	}

	slog.Debug("Coalescing request...")
	fetchedObj, err, shared := f.group.Do(key.Hex, func() (any, error) {
		return f.internalDedupFetch(req, key)
	})
	if err != nil {
		return fetchResult{}, err
	}

	// If shared is true, this means the request was coalesced.
	// Meaning that the result is potentially being shared with other requests.

	if shared {
		metrics.Global.Requests.CoalescedRequests.Increment()
	}

	fetched = fetchedObj.(fetchResult)

	switch fetched.Type {
	case fetchTypeCached:
		slog.Debug("Fetched cached response", "url", req.URL, "status", fetched.Cached.Status, "upstream_status", fetched.Cached.UpstreamStatus, "coalesced", shared)
		fetched.Cached.Coalesced = shared

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

		fetched.Cached.Entry, err = f.cache.Get(key)
		if err != nil {
			if errors.Is(err, cache.ErrCacheMiss) {
				return f.handleCacheMiss(req, key)
			}
			slog.Error("Error getting newly cached response. Bypassing cache and fetching upstream...", "url", req.URL, "key", key, "error", err)
			return f.fetchUpstream(req)
		}
		return fetched, nil

	case fetchTypeDirect:
		if shared {
			slog.Debug("Fetched shared direct response, fetching own upstream...", "url", req.URL, "status", fetched.Direct.Status, "upstream_status", fetched.Direct.UpstreamStatus, "coalesced", shared)
			return f.fetchUpstream(req) // Followers should fetch their own upstream response
		}
		slog.Debug("Fetched direct response", "url", req.URL, "status", fetched.Direct.Status, "upstream_status", fetched.Direct.UpstreamStatus, "coalesced", shared)
		return fetched, nil

	default:
		// This should not be possible unless new fetch types are introduced and not correctly implemented.
		slog.Error("Unknown fetch result type", "type", fetched.Type)
		return fetchResult{}, errors.ErrUnsupported
	}
}
