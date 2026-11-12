package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/headers"
	"reservoir/utils/syncmap"
	"strings"
)

var errNotCacheable = errors.New("response not cacheable")

type Fetcher struct {
	cache        cache.Cache[CacheMetadata]
	cfg          *config.Config
	metrics      *metrics.Metrics
	policy       cachePolicy
	client       *http.Client
	flights      flightGroup
	liveWindow   int // See defaultLiveWindow.
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
		liveWindow:   defaultLiveWindow,
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

	fl := f.flights.join(f.singleflightKey(req, baseKey), func(fl *flight) (Result, error) {
		// Others may be waiting on this download, so the first client leaving must not cancel it.
		detached := req.WithContext(withFlight(context.WithoutCancel(req.Context()), fl))
		fetched, err := f.getFromCacheOrFetch(detached, baseKey, lookupKey, clientHd)
		if err == nil && fl.liveBody() != nil && fetched.Type == TypeCached && fetched.Cached.Entry.Data != nil {
			// Requests read the live body or open the entry themselves, so nobody else would close this handle.
			fetched.Cached.Entry.Data.Close()
		}
		return fetched, err
	})

	select {
	case <-fl.started:
	case <-fl.done:
	}
	// A body that is still downloading is read as it arrives, not after the whole file is cached.
	if body := fl.liveBody(); body != nil {
		if fetched, ok := f.liveResult(req, fl, body); ok {
			return fetched, nil
		}
	}

	<-fl.done
	fetched, err = fl.result, fl.err
	shared := f.flights.shared(fl)
	// After a live body the flight has closed its own handle on the entry.
	ownHandle := shared || fl.liveBody() != nil
	if err != nil {
		if errors.Is(err, errNotCacheable) {
			slog.Debug("Request was not cacheable in singleflight, falling back to direct fetch", "url", req.URL)
			f.metrics.Requests.NonCoalescedRequests.Increment()
			return f.fetchDirectlyFromUpstream(req)
		}
		f.countRequest(shared)
		return Result{}, err
	}

	f.countRequest(shared)

	switch fetched.Type {
	case TypeCached:
		fetched.Cached.Coalesced = shared

		// If the result is shared, we cannot share the same Data (seeker) between goroutines.
		// We close the shared handle and each caller gets their own fresh one.
		if ownHandle {
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

// liveResult answers a request from a body that is still being downloaded. It
// reports false if the start of the body is no longer available to read along.
func (f *Fetcher) liveResult(req *http.Request, fl *flight, body *liveBody) (Result, bool) {
	reader := body.newReader(func(off int64, cached bool) (io.ReadCloser, error) {
		return f.resumeBody(req, body, off, cached)
	})
	if reader == nil {
		return Result{}, false
	}

	shared := f.flights.shared(fl)
	f.countRequest(shared)
	if shared {
		switch fl.bodyStatus {
		case StatusRevalidated:
			f.metrics.Requests.CoalescedCacheRevalidations.Increment()
		default:
			f.metrics.Requests.CoalescedCacheMisses.Increment()
		}
	}

	entry := body.entry
	fetched := newCachedResult(&cache.Entry[CacheMetadata]{Data: reader, Metadata: &entry}, fl.bodyStatus, http.StatusOK, 0)
	fetched.Cached.Coalesced = shared
	slog.Debug("Reading response from running download", "url", req.URL, "status", fl.bodyStatus, "coalesced", shared)
	return fetched, true
}

// resumeBody opens the rest of a body for a request that could not read all of
// it from the running download: from the cache if the download made it there,
// otherwise from upstream with a range request.
func (f *Fetcher) resumeBody(req *http.Request, body *liveBody, off int64, cached bool) (io.ReadCloser, error) {
	if cached {
		entry, err := f.cache.Get(body.key)
		if err == nil {
			if _, err = entry.Data.Seek(off, io.SeekStart); err == nil {
				return entry.Data, nil
			}
			entry.Data.Close()
		}
		slog.Warn("Could not continue response from the cache, continuing from upstream", "url", req.URL, "error", err)
	}

	object := body.entry.Object
	up := req.Clone(req.Context())
	up.Header.Set("Range", fmt.Sprintf("bytes=%d-", off))
	// Upstream must only continue if it still has the same file.
	if object.ETag != "" {
		up.Header.Set("If-Range", object.ETag)
	} else {
		up.Header.Set("If-Range", object.LastModified.Format(http.TimeFormat))
	}

	resp, _, err := f.sendRequestToUpstream(up)
	if err != nil {
		return nil, err
	}
	rest := trackFetchedBytes(resp.Body, f.metrics)

	switch {
	case resp.StatusCode == http.StatusPartialContent && strings.HasPrefix(resp.Header.Get("Content-Range"), fmt.Sprintf("bytes %d-", off)):
		return rest, nil
	case resp.StatusCode == http.StatusOK && sameFile(resp.Header, object.Header):
		// Upstream does not do ranges and sent the same file from its first byte: skip what the client has.
		if _, err := io.CopyN(io.Discard, rest, off); err == nil {
			return rest, nil
		}
	}
	rest.Close()
	return nil, fmt.Errorf("%w: upstream did not continue at byte %d (status %d)", errDownloadIncomplete, off, resp.StatusCode)
}

// sameFile reports whether two responses carry the same validator, so their bodies can be joined.
func sameFile(a http.Header, b http.Header) bool {
	for _, validator := range []string{"ETag", "Last-Modified"} {
		if value := a.Get(validator); value != "" {
			return value == b.Get(validator)
		}
	}
	return false
}
