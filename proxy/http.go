package proxy

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/proxy/fetch"
	"reservoir/proxy/headers"
	"reservoir/proxy/responder"
	"reservoir/utils/typeutils"
	"time"
)

func (p *Proxy) ServeHTTP(w http.ResponseWriter, proxyReq *http.Request) {
	r := responder.NewHTTPResponder(w)
	if proxyReq.Method == http.MethodConnect {
		if err := p.handleCONNECT(r, proxyReq); err != nil {
			slog.Error("Error handling CONNECT request", "error", err)
			return
		}
	} else {
		if err := p.handleHTTP(r, proxyReq); err != nil {
			slog.Error("Error handling HTTP request", "error", err)
			return
		}
	}
}

func (p *Proxy) processRequest(r responder.Responder, req *http.Request, key cache.CacheKey, clientHd *headers.HeaderDirectives) error {
	slog.Debug("Processing HTTP request", "remote_addr", req.RemoteAddr, "method", req.Method, "url", req.URL)

	startTime := time.Now()

	defer func() {
		p.metrics.Requests.ClientRequestLatency.Add(time.Since(startTime).Nanoseconds())
	}()

	fetched, err := p.fetcher.Fetch(req, key, clientHd)
	latency := time.Since(startTime)

	if err != nil {
		slog.Error("Error fetching resource", "url", req.URL, "key", key, "error", err)
		r.WriteError("Error fetching resource", http.StatusBadGateway)
		return err
	}

	fetchInfo := fetched.FetchInfo()
	switch fetchInfo.Status {
	case fetch.StatusMiss:
		p.metrics.Cache.CacheRequestMisses.Increment()
		p.metrics.Cache.CacheMissLatency.Add(latency.Nanoseconds())
	case fetch.StatusRevalidated:
		p.metrics.Cache.CacheRequestRevalidations.Increment()
		p.metrics.Cache.CacheHitLatency.Add(latency.Nanoseconds())
	case fetch.StatusStale:
		p.metrics.Cache.CacheRequestStales.Increment()
		p.metrics.Cache.CacheHitLatency.Add(latency.Nanoseconds())
	default:
		p.metrics.Cache.CacheRequestHits.Increment()
		p.metrics.Cache.CacheHitLatency.Add(latency.Nanoseconds())
	}

	switch fetched.Type {
	case fetch.TypeDirect:
		defer fetched.Direct.Response.Body.Close()

		r.SetHeaders(fetched.Direct.Response.Header)
		if fetched.Direct.UpstreamStatus >= 200 && fetched.Direct.UpstreamStatus < 300 {
			r.SetHeader("Accept-Ranges", "bytes")
			addCacheHeaders(r, req, typeutils.None[*cache.Entry[fetch.CacheMetadata]](), fetchResultToCacheStatus(fetched))
		}

		return finalizeAndRespond(p.metrics, r, fetched.Direct.Response.Body, fetched.Direct.UpstreamStatus, req)

	case fetch.TypeCached:
		if fetched.Cached.Entry == nil || fetched.Cached.Entry.Data == nil {
			slog.Error("fetch.TypeCached: entry or data is nil", "url", req.URL)
			r.WriteError("internal error: cache entry data is nil", http.StatusInternalServerError)
			return fmt.Errorf("cache entry data is nil")
		}
		defer fetched.Cached.Entry.Data.Close()

		if clientHd.Range.IsPresent() {
			if err := p.handleRangeRequest(r, req, fetched.Cached.Entry, key, clientHd); err != nil {
				slog.Error("Error handling Range request", "url", req.URL, "key", key, "error", err)
				if errors.Is(err, ErrIfRangeMismatch) {
					// If the If-Range is mismatched we just move on to send the full 200 cached response.
				} else if errors.Is(err, ErrRangeNotSatisfiable) {
					return err
				} else {
					r.WriteError("error handling Range request", http.StatusInternalServerError)
					return err
				}
			} else {
				return nil
			}
		}

		r.SetHeaders(fetched.Cached.Entry.Metadata.Object.Header)
		r.SetHeader("Accept-Ranges", "bytes")
		r.SetHeader("ETag", fetched.Cached.Entry.Metadata.Object.ETag)
		r.SetHeader("Last-Modified", fetched.Cached.Entry.Metadata.Object.LastModified.Format(http.TimeFormat))
		addCacheHeaders(r, req, typeutils.Some(fetched.Cached.Entry), fetchResultToCacheStatus(fetched))

		slog.Debug("Serving cached response", "url", req.URL, "key", key)
		return finalizeAndRespond(p.metrics, r, fetched.Cached.Entry.Data, http.StatusOK, req)

	default:
		// This should not be possible
		return fmt.Errorf("unknown fetch type: %v", fetched.Type)
	}
}

func (p *Proxy) handleHTTP(r responder.Responder, proxyReq *http.Request) error {
	slog.Debug("Handling HTTP request", "host", proxyReq.Host, "remote_addr", proxyReq.RemoteAddr)
	p.metrics.Requests.HTTPProxyRequests.Increment()

	clientHd := headers.ParseHeaderDirective(proxyReq.Header)
	clientHd.StripRegularConditionals(proxyReq.Header)

	key := cache.MakeFromRequest(proxyReq)

	return p.processRequest(r, proxyReq, key, clientHd)
}
