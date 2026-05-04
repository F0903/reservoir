package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/metrics"
	"reservoir/proxy/headers"
	"reservoir/utils/countingreader"
	"time"
)

func (f *fetcher) handleUpstream200(req *http.Request, resp *http.Response, baseKey cache.CacheKey, lookupKey cache.CacheKey, upstreamHd *headers.HeaderDirectives) (cached *cache.Entry[cachedRequestInfo], err error) {
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
		return f.handleUpstream200(req, resp, baseKey, lookupKey, upstreamHd)
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
