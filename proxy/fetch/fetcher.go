package fetch

import (
	"context"
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

var errNotCacheable = errors.New("response not cacheable")

type Fetcher struct {
	cache        cache.Cache[CacheMetadata]
	cfg          *config.Config
	metrics      *metrics.Metrics
	policy       cachePolicy
	client       *http.Client
	group        singleflight.Group
	variantIndex *syncmap.SyncMap[cache.CacheKey, []string]
}

func New(cacheStore cache.Cache[CacheMetadata], cfg *config.Config, upstreamClient *http.Client, m *metrics.Metrics) *Fetcher {
	if upstreamClient == nil {
		upstreamClient = newUpstreamClient()
	}
	m = metrics.Ensure(m)

	return &Fetcher{
		cache:        cacheStore,
		cfg:          cfg,
		metrics:      m,
		policy:       newCachePolicy(cfg),
		client:       upstreamClient,
		group:        singleflight.Group{},
		variantIndex: syncmap.New[cache.CacheKey, []string](),
	}
}

func (f *Fetcher) CloseIdleConnections() {
	f.client.CloseIdleConnections()
}

// countRequest records whether the result was shared with other requests in-flight.
func (f *Fetcher) countRequest(shared bool) {
	if shared {
		f.metrics.Requests.CoalescedRequests.Increment()
	} else {
		f.metrics.Requests.NonCoalescedRequests.Increment()
	}
}

// Fetch deduplicates cacheable requests and otherwise returns the bypassed upstream response.
// IMPORTANT: Remember to close data streams!
func (f *Fetcher) Fetch(req *http.Request, baseKey cache.CacheKey, clientHd *headers.HeaderDirectives) (fetched Result, err error) {
	slog.Debug("Attempting to dedup fetch...")
	lookupKey := f.lookupCacheKey(req, baseKey)

	if !f.policy.RequestAllowsSharedCache(req) {
		slog.Debug("Request contains credentials, bypassing shared cache", "url", req.URL)
		f.metrics.Requests.NonCoalescedRequests.Increment()
		return f.fetchDirectlyFromUpstream(req)
	}

	shouldCoalesce := !clientHd.Range.IsPresent() && req.Method == http.MethodGet
	if !shouldCoalesce {
		f.metrics.Requests.NonCoalescedRequests.Increment()

		if req.Method == http.MethodGet {
			if cached := f.getFreshEntry(lookupKey); cached != nil {
				slog.Debug("Range request found a fresh cache entry", "url", req.URL, "key", lookupKey)
				return newCachedResult(cached, StatusHit, 0, 0), nil
			}
		}

		// These requests also aren't cacheable, so they go straight to upstream.
		slog.Debug("Request can't be coalesced, fetching upstream...")
		fetched, err := f.fetchUpstream(req, baseKey, lookupKey, clientHd)
		if errors.Is(err, errNotCacheable) {
			return f.fetchDirectlyFromUpstream(req)
		}
		return fetched, err
	}

	fetchedObj, err, shared := f.group.Do(f.singleflightKey(req, baseKey), func() (any, error) {
		// Others may be waiting on this download, so the first client leaving must not cancel it.
		detached := req.WithContext(context.WithoutCancel(req.Context()))
		return f.getFromCacheOrFetch(detached, baseKey, lookupKey, clientHd)
	})
	if err != nil {
		if errors.Is(err, errNotCacheable) {
			slog.Debug("Request was not cacheable in singleflight, falling back to direct fetch", "url", req.URL)
			f.metrics.Requests.NonCoalescedRequests.Increment()
			return f.fetchDirectlyFromUpstream(req)
		}
		f.countRequest(shared)
		return Result{}, err
	}
	fetched = fetchedObj.(Result)

	f.countRequest(shared)

	switch fetched.Type {
	case TypeCached:
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
			case StatusHit:
				f.metrics.Requests.CoalescedCacheHits.Increment()
			case StatusRevalidated:
				f.metrics.Requests.CoalescedCacheRevalidations.Increment()
			case StatusMiss:
				f.metrics.Requests.CoalescedCacheMisses.Increment()
			case StatusStale:
				f.metrics.Requests.CoalescedCacheStales.Increment()
			}
		}

		slog.Debug("Fetched cached response", "url", req.URL, "status", fetched.Cached.Status, "upstream_status", fetched.Cached.UpstreamStatus, "coalesced", shared)
		return fetched, nil

	case TypeDirect:
		if shared {
			if fetched.Direct.Response != nil && fetched.Direct.Response.Body != nil {
				fetched.Direct.Response.Body.Close()
			}
			slog.Debug("Fetched shared direct response, fetching own upstream...", "url", req.URL, "status", fetched.Direct.Status, "upstream_status", fetched.Direct.UpstreamStatus, "coalesced", shared)
			return f.fetchDirectlyFromUpstream(req) // Followers should fetch their own upstream response
		}
		slog.Debug("Fetched direct response", "url", req.URL, "status", fetched.Direct.Status, "upstream_status", fetched.Direct.UpstreamStatus, "coalesced", shared)
		return fetched, nil

	default:
		// This should not be possible unless new fetch types are introduced and not correctly implemented.
		slog.Error("Unknown fetch result type", "type", fetched.Type)
		return Result{}, errors.ErrUnsupported
	}
}
