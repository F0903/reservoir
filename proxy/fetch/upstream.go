package fetch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/proxy/headers"
	"reservoir/utils/countingreader"
	"time"
)

func (f *Fetcher) shouldBypassCache(resp *http.Response) bool {
	stats := f.cache.Stats()
	if stats.MaxBytes <= 0 {
		return true
	}
	if resp.ContentLength > stats.MaxBytes {
		return true
	}

	switch f.cfg.Cache.Type.Read() {
	case config.CacheTypeMemory:
		if resp.ContentLength < 0 {
			return true
		}

		limit := f.memoryCacheResponseLimit()
		return limit <= 0 || resp.ContentLength > limit
	case config.CacheTypeFile, config.CacheTypeHybrid:
		// Unknown-size file-tier writes cannot be safely replayed to the client if
		// they exceed the hard cache limit mid-stream, so serve them directly.
		return resp.ContentLength < 0
	default:
		return true
	}
}

func (f *Fetcher) memoryCacheResponseLimit() int64 {
	stats := f.cache.Stats()
	if stats.MaxBytes <= 0 {
		return stats.MaxBytes
	}
	if stats.MemoryCapBytes <= 0 {
		return stats.MemoryCapBytes
	}
	return min(stats.MaxBytes, stats.MemoryCapBytes)
}

func (f *Fetcher) handleUpstream200(req *http.Request, resp *http.Response, baseKey cache.CacheKey, lookupKey cache.CacheKey, upstreamHd *headers.HeaderDirectives) (cached *cache.Entry[CacheMetadata], err error) {
	slog.Debug("Handling 200 response from upstream", "url", req.URL, "key", lookupKey)

	decision := f.policy.Decide(req, resp, upstreamHd)
	if !decision.Cacheable {
		slog.Debug("Got response code 200, but result is not cacheable", "status", resp.Status, "url", req.URL, "key", lookupKey, "reason", decision.Reason)
		return nil, nil
	}
	if f.shouldBypassCache(resp) {
		slog.Debug("Skipping cache for response that cannot be safely stored", "status", resp.Status, "url", req.URL, "key", lookupKey, "content_length", resp.ContentLength)
		return nil, nil
	}

	storeKey := makeVariantCacheKey(req, baseKey, decision.Vary)

	slog.Debug("Caching response...", "status", resp.Status, "url", req.URL, "key", storeKey, "lookup_key", lookupKey)

	lastModified := time.Now()
	if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
		lastModified = t
	}

	metadata := CacheMetadata{
		ETag:         resp.Header.Get("ETag"),
		LastModified: lastModified,
		Header:       resp.Header,
		Vary:         decision.Vary,
	}

	var bytesRead int
	var body io.Reader = countingreader.New(resp.Body, &bytesRead)

	// Requests waiting on this flight read the body from a live copy while the cache
	// is still downloading it, so they are not kept waiting for the whole file.
	if fl := flightFrom(req.Context()); fl != nil {
		now := time.Now()
		live := &liveSource{
			src:   body,
			start: fl.startBody,
			body: newLiveBody(storeKey, cache.EntryMetadata[CacheMetadata]{
				TimeWritten: now,
				LastAccess:  now,
				Expires:     decision.Expires,
				Size:        resp.ContentLength,
				Object:      metadata,
			}, f.liveWindow),
		}
		body = live
		defer func() { live.settle(err == nil && cached != nil) }()
	}

	cached, err = f.cache.Cache(storeKey, cache.WithSizeHint(body, resp.ContentLength), decision.Expires, metadata)
	f.metrics.Requests.BytesFetched.Add(int64(bytesRead))
	if err != nil {
		// The cache can't hold this response, but the client can still get it directly.
		return nil, fmt.Errorf("%w: %v", errNotCacheable, err)
	}
	f.setVariantIndex(baseKey, decision.Vary)

	slog.Info("Successfully cached response", "status", resp.Status, "url", req.URL, "key", storeKey, "expires", decision.Expires)
	return cached, nil
}

func (f *Fetcher) handleUpstream416(req *http.Request, resp *http.Response, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives, noRetry bool) (cached *cache.Entry[CacheMetadata], err error) {
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

func (f *Fetcher) handleUpstreamResponse(req *http.Request, resp *http.Response, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives, noRetry bool) (cached *cache.Entry[CacheMetadata], err error) {
	slog.Debug("Preparing to handle upstream response...", "url", req.URL, "status", resp.StatusCode)

	upstreamHd := headers.ParseHeaderDirective(resp.Header)

	switch resp.StatusCode {
	case http.StatusOK:
		return f.handleUpstream200(req, resp, baseKey, lookupKey, upstreamHd)
	case http.StatusNotModified:
		return f.handleUpstream304(req, lookupKey, upstreamHd)
	case http.StatusRequestedRangeNotSatisfiable:
		return f.handleUpstream416(req, resp, baseKey, lookupKey, clientHd, noRetry)
	default:
		slog.Debug("Upstream returned non-cacheable response", "url", req.URL, "status", resp.StatusCode)
		return nil, nil
	}
}

func (f *Fetcher) sendRequestToUpstream(req *http.Request) (*http.Response, time.Duration, error) {
	slog.Debug("Sending request to upstream", "url", req.URL)
	f.metrics.Requests.UpstreamRequests.Increment()

	ctx, cancel := context.WithCancel(req.Context())

	startTime := time.Now()
	resp, err := sendRequestToTarget(f.client, req.WithContext(ctx), f.cfg.Proxy.UpgradeHttpToHttps.Read())
	latency := time.Since(startTime)

	f.metrics.Requests.UpstreamRequestLatency.Add(latency.Nanoseconds())
	if err != nil {
		cancel()
		return nil, 0, err
	}
	resp.Body = newIdleTimeoutBody(resp.Body, upstreamBodyIdleTimeout, cancel)

	slog.Debug("Received response from upstream", "url", req.URL, "status", resp.Status, "latency_ns", latency.Nanoseconds())
	return resp, latency, nil
}

func (f *Fetcher) fetchUpstream(req *http.Request, baseKey cache.CacheKey, lookupKey cache.CacheKey, clientHd *headers.HeaderDirectives) (Result, error) {
	slog.Debug("Fetching from upstream...")

	resp, upstreamLatency, err := f.sendRequestToUpstream(req)
	if err != nil {
		slog.Error("Error fetching upstream", "url", req.URL, "error", err)
		return Result{}, err
	}

	cached, err := f.handleUpstreamResponse(req, resp, baseKey, lookupKey, clientHd, false)
	if err != nil {
		resp.Body.Close()
		slog.Error("Error handling upstream response after cache miss", "url", req.URL, "error", err)
		return Result{}, err
	}

	if cached == nil {
		slog.Debug("Upstream response is not cacheable, returning direct result after cache miss...", "url", req.URL, "status", resp.StatusCode)

		resp.Body = trackFetchedBytes(resp.Body, f.metrics)

		return newDirectResult(resp, StatusMiss, upstreamLatency), nil
	}

	// We have a cached entry, so we don't need the original upstream response body anymore.
	resp.Body.Close()

	slog.Debug("Returning cached fetch result...")

	return newCachedResult(cached, StatusMiss, resp.StatusCode, upstreamLatency), nil
}

// Fetches directly from upstream without caching.
// Used for requests where you don't care about caching, and just want to pass it straight back to the client.
func (f *Fetcher) fetchDirectlyFromUpstream(req *http.Request) (Result, error) {
	slog.Debug("Fetching directly from upstream...")
	resp, upstreamLatency, err := f.sendRequestToUpstream(req)
	if err != nil {
		slog.Error("Error fetching upstream", "url", req.URL, "error", err)
		return Result{}, err
	}

	resp.Body = trackFetchedBytes(resp.Body, f.metrics)

	slog.Debug("Returning direct fetch result", "url", req.URL, "status", resp.StatusCode)
	return newDirectResult(resp, StatusMiss, upstreamLatency), nil
}
