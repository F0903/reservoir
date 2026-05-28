package fetch

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/proxy/headers"
	"time"
)

func (f *Fetcher) handleUpstream304(req *http.Request, key cache.CacheKey) (cached *cache.Entry[CacheMetadata], err error) {
	slog.Debug("Handling 304 response from upstream", "url", req.URL, "key", key)

	slog.Debug("Revalidating cache metadata...", "url", req.URL, "key", key)
	err = f.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[CacheMetadata]) {
		// Update the metadata to reflect that the cached response is still valid.
		maxAge := f.cfg.Proxy.CachePolicy.DefaultMaxAge.Read().Cast()
		meta.Expires = time.Now().Add(maxAge)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errUpdateCacheMetadata, err)
	}

	slog.Debug("Successfully revalidated cache metadata", "url", req.URL, "key", key)
	return f.cache.Get(key)
}

func (f *Fetcher) serveStaleCachedResponse(req *http.Request, lookupKey cache.CacheKey, upstreamStatus int, upstreamErr error) (Result, error) {
	if upstreamErr != nil {
		slog.Warn("Serving stale cached response because upstream revalidation failed", "url", req.URL, "key", lookupKey, "error", upstreamErr)
	} else {
		slog.Warn("Serving stale cached response because upstream returned an error status", "url", req.URL, "key", lookupKey, "upstream_status", upstreamStatus)
	}

	cached, err := f.cache.Get(lookupKey)
	if err != nil {
		if upstreamErr != nil {
			return Result{}, upstreamErr
		}
		return Result{}, err
	}

	return newCachedResult(cached, StatusStale, upstreamStatus, 0), nil
}

// Fetches the requested resource either from cache or upstream.
func (f *Fetcher) getFromCacheOrFetch(req *http.Request, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives) (Result, error) {
	slog.Debug("Trying to get request from cache...")

	cached, err := f.cache.Get(lookupKey)
	if err != nil {
		if errors.Is(err, cache.ErrCacheEntryNotFound) {
			slog.Debug("Cache miss, will fetch from upstream.", "url", req.URL, "key", lookupKey)
			res, err := f.fetchUpstream(req, baseKey, lookupKey, clientHd)
			if err != nil {
				return Result{}, err
			}
			return res, nil
		}

		f.metrics.Cache.CacheErrors.Increment()
		// We just return errNotCacheable to trigger direct fetch in caller
		return Result{}, errNotCacheable
	}

	if !cached.Stale {
		slog.Debug("Cache hit, returning cached response.", "url", req.URL, "key", lookupKey)
		return newCachedResult(cached, StatusHit, 0, 0), nil
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

	fetched, err := f.fetchUpstream(up, baseKey, lookupKey, clientHd)
	if err != nil {
		return f.serveStaleCachedResponse(req, lookupKey, 0, err)
	}
	if fetched.Type == TypeDirect {
		if fetched.Direct.UpstreamStatus >= 500 {
			status := fetched.Direct.UpstreamStatus
			fetched.Direct.Response.Body.Close()
			return f.serveStaleCachedResponse(req, lookupKey, status, nil)
		}

		// The upstream response is authoritative but not cacheable. Return it
		// directly instead of retrying a second upstream request.
		slog.Debug("Stale revalidation returned a non-cacheable direct response", "url", req.URL, "status", fetched.Direct.UpstreamStatus)
		return fetched, nil
	}
	fetched.Cached.Status = StatusRevalidated

	return fetched, nil
}
