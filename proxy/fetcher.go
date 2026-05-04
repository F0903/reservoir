package proxy

import (
	"errors"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/headers"
	"reservoir/utils/syncmap"

	"golang.org/x/sync/singleflight"
)

var ErrNotCacheable = errors.New("response not cacheable")

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

func (f *fetcher) closeIdleConnections() {
	f.client.CloseIdleConnections()
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
