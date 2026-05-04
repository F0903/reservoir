package proxy

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/metrics"
	"reservoir/proxy/headers"
	"time"
)

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
