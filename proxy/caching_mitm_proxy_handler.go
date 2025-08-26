package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/certs"
	"reservoir/proxy/headers"
	"reservoir/proxy/responder"
	"time"
)

var (
	ErrCacheGetFailed       = errors.New("error getting cache for key")
	ErrNoCachedResponse304  = errors.New("received 304 Not Modified but no cached response found")
	ErrUpdateCacheMetadata  = errors.New("error updating cache metadata")
	ErrCacheResponseFailed  = errors.New("error caching response")
	ErrHijackNotSupported   = errors.New("hijacking not supported for target host")
	ErrHijackFailed         = errors.New("hijack failed")
	ErrTLSCertFailed        = errors.New("error getting TLS certificate")
	ErrClientResponseFailed = errors.New("failed to write HTTP OK response to client")
	ErrReadRequestFailed    = errors.New("error reading request from client")
	ErrRangeNotSatisfiable  = errors.New("range not satisfiable")
	ErrIfRangeMismatch      = errors.New("If-Range header mismatch")
	ErrBadGateway           = errors.New("bad gateway. Error when sending request to upstream")
)

type cachedRequestInfo struct {
	ETag         string
	LastModified time.Time
	Header       http.Header
}

type cachingMitmProxyHandler struct {
	ca    certs.CertAuthority
	cache *cache.FileCache[cachedRequestInfo]
}

func newCachingMitmProxyHandler(cacheDir string, ca certs.CertAuthority, ctx context.Context) (*cachingMitmProxyHandler, error) {
	cfgLock := config.Global.Immutable()

	var cacheCleanupInterval time.Duration
	cfgLock.Read(func(c *config.Config) {
		cacheCleanupInterval = c.CacheCleanupInterval.Read().Cast()
	})

	return &cachingMitmProxyHandler{
		ca:    ca,
		cache: cache.NewFileCache[cachedRequestInfo](cacheDir, cacheCleanupInterval, ctx),
	}, nil
}

func (p *cachingMitmProxyHandler) ServeHTTP(w http.ResponseWriter, proxyReq *http.Request) {
	if proxyReq.Method == http.MethodConnect {
		if err := p.handleCONNECT(w, proxyReq); err != nil {
			slog.Error("Error handling CONNECT request", "error", err)
			return
		}
	} else {
		if err := p.handleHTTP(w, proxyReq); err != nil {
			slog.Error("Error handling HTTP request", "error", err)
			return
		}
	}
}

func shouldResponseBeCached(resp *http.Response, upstreamHd *headers.HeaderDirectives) bool {
	cfgLock := config.Global.Immutable()

	var alwaysCache bool
	cfgLock.Read(func(c *config.Config) {
		alwaysCache = c.AlwaysCache.Read()
	})

	if alwaysCache {
		return true
	}

	return upstreamHd.ShouldCache() &&
		resp.StatusCode == http.StatusOK &&
		resp.Request.Method == http.MethodGet
}

func sendResponse(r responder.Responder, resp io.Reader, status int, req *http.Request) error {
	body := resp
	if req.Method == http.MethodHead {
		body = http.NoBody
	}

	written, err := r.Write(status, body)
	if err != nil {
		slog.Error("Error writing response", "url", req.URL, "error", err)
		return err
	}

	metrics.Global.Requests.BytesServed.Add(written)
	slog.Info("Response sent", "url", req.URL, "bytes_written", written)
	return nil
}

func (p *cachingMitmProxyHandler) handleUpstream304(r responder.Responder, req *http.Request, resp *http.Response, cached *cache.Entry[cachedRequestInfo], key cache.CacheKey, clientHd *headers.HeaderDirectives) error {
	if cached == nil {
		// This should not be possible
		slog.Error("Received 304 Not Modified but no cached response found", "url", req.URL, "key", key, "headers", req.Header)
		r.WriteError("malformed state", http.StatusInternalServerError)
		return ErrNoCachedResponse304
	}

	var defaultCacheMaxAge time.Duration
	cfgLock := config.Global.Immutable()
	cfgLock.Read(func(c *config.Config) {
		defaultCacheMaxAge = c.DefaultCacheMaxAge.Read().Cast()
	})

	err := p.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[cachedRequestInfo]) {
		// Update the metadata to reflect that the cached response is still valid.
		maxAge := defaultCacheMaxAge
		meta.Expires = time.Now().Add(maxAge)
	})
	if err != nil {
		slog.Error("Error updating cache metadata", "url", req.URL, "key", key, "error", err)
		r.WriteError("error updating cache metadata", http.StatusInternalServerError)
		return fmt.Errorf("%w: %v", ErrUpdateCacheMetadata, err)
	}

	slog.Info("Origin server returned 304 Not Modified, serving cached response", "url", req.URL, "key", key)

	if clientHd.Range.IsSome() {
		if err := p.handleRangeRequest(r, req, cached, key, clientHd); err != nil {
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

	r.SetHeaders(cached.Metadata.Object.Header)
	r.SetHeader("Accept-Ranges", "bytes")
	r.SetHeader("ETag", cached.Metadata.Object.ETag)
	r.SetHeader("Last-Modified", cached.Metadata.Object.LastModified.Format(http.TimeFormat))

	return sendResponse(r, cached.Data, http.StatusOK, req)
}

func (p *cachingMitmProxyHandler) handleUpstream206(r responder.Responder, req *http.Request, resp *http.Response) error {
	r.SetHeaders(resp.Header)
	r.SetHeader("Accept-Ranges", "bytes")

	slog.Info("Origin server returned 206 Partial Content, serving response", "url", req.URL)
	return sendResponse(r, resp.Body, http.StatusPartialContent, req)
}

func (p *cachingMitmProxyHandler) handleUpstream200(r responder.Responder, req *http.Request, resp *http.Response, key cache.CacheKey, upstreamHd *headers.HeaderDirectives) error {
	var data io.Reader = resp.Body

	if shouldResponseBeCached(resp, upstreamHd) {
		slog.Info("Caching response", "status", resp.Status, "url", req.URL, "key", key)

		lastModified := time.Now()
		if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
			lastModified = t
		}

		etag := resp.Header.Get("ETag")

		maxAge := upstreamHd.GetExpiresOrDefault()
		entry, err := p.cache.Cache(key, resp.Body, maxAge, cachedRequestInfo{
			ETag:         etag,
			LastModified: lastModified,
			Header:       resp.Header,
		})
		if err != nil {
			slog.Error("Error caching response", "url", req.URL, "key", key, "error", err)
			r.WriteError("error caching response", http.StatusInternalServerError)
			return fmt.Errorf("%w: %v", ErrCacheResponseFailed, err)
		}
		defer entry.Data.Close() // Ensure we close the cached data when done

		data = entry.Data
	}

	r.SetHeaders(resp.Header)
	r.SetHeader("Accept-Ranges", "bytes")

	slog.Info("Sending response", "url", req.URL, "status", resp.StatusCode)
	return sendResponse(r, data, http.StatusOK, req)
}

func (p *cachingMitmProxyHandler) handleUpstreamResponse(r responder.Responder, req *http.Request, resp *http.Response, cached *cache.Entry[cachedRequestInfo], key cache.CacheKey, clientHd *headers.HeaderDirectives) error {
	slog.Debug("Handling upstream response...", "url", req.URL, "status", resp.StatusCode)

	upstreamHd := headers.ParseHeaderDirective(resp.Header)

	switch resp.StatusCode {
	case http.StatusOK:
		return p.handleUpstream200(r, req, resp, key, upstreamHd)
	case http.StatusNotModified:
		return p.handleUpstream304(r, req, resp, cached, key, clientHd)
	case http.StatusPartialContent:
		return p.handleUpstream206(r, req, resp)
	default:
		slog.Info("Upstream returned non-cachable response, forwarding as-is", "url", req.URL, "status", resp.StatusCode)

		r.SetHeaders(resp.Header)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			r.SetHeader("Accept-Ranges", "bytes")
		}
		return sendResponse(r, resp.Body, resp.StatusCode, req)
	}
}

func (p *cachingMitmProxyHandler) handleRangeRequest(r responder.Responder, req *http.Request, cached *cache.Entry[cachedRequestInfo], key cache.CacheKey, clientHd *headers.HeaderDirectives) error {
	rangeHeader := clientHd.Range.ForceUnwrap() // hd.Range will always be Some here
	start, end, err := rangeHeader.SliceSize(cached.Metadata.FileSize)
	if err != nil {
		slog.Error("Error parsing Range header", "url", req.URL, "key", key, "error", err)

		r.SetHeader("Content-Range", fmt.Sprintf("bytes */%d", cached.Metadata.FileSize))
		r.WriteError("invalid Range header", http.StatusRequestedRangeNotSatisfiable)

		return ErrRangeNotSatisfiable
	}

	if ifRangeOpt := clientHd.ConditionalHeaders.IfRange; ifRangeOpt.IsSome() {
		ifRange := ifRangeOpt.ForceUnwrap()
		if ifRange.IsLeft() {
			// IfRange is ETag
			etagIfRange := ifRange.ForceUnwrapLeft()
			if etagIfRange != cached.Metadata.Object.ETag {
				slog.Info("If-Range does not match cached ETag. Sending full 200 response.", "url", req.URL, "key", key)
				return ErrIfRangeMismatch
			}
		} else {
			// IfRange is Time
			timeIfRange := ifRange.ForceUnwrapRight()
			if timeIfRange.Before(cached.Metadata.Object.LastModified) {
				slog.Info("If-Range does not match cached Last-Modified. Sending full 200 response.", "url", req.URL, "key", key)
				return ErrIfRangeMismatch
			}
		}

	}

	slog.Info("Serving Range request from cache", "url", req.URL, "key", key, "range_header", rangeHeader, "start", start, "end", end)

	length := end - start + 1

	r.SetHeaders(cached.Metadata.Object.Header)
	r.SetHeader("Accept-Ranges", "bytes")
	r.SetHeader("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, cached.Metadata.FileSize))
	r.SetHeader("Content-Length", fmt.Sprintf("%d", length))
	r.SetHeader("ETag", cached.Metadata.Object.ETag)
	r.SetHeader("Last-Modified", cached.Metadata.Object.LastModified.Format(http.TimeFormat))

	sections := io.NewSectionReader(cached.Data, start, length)
	return sendResponse(r, sections, http.StatusPartialContent, req)
}

func (p *cachingMitmProxyHandler) handleCachedResponse(r responder.Responder, req *http.Request, cached *cache.Entry[cachedRequestInfo], key cache.CacheKey, clientHd *headers.HeaderDirectives) error {
	slog.Debug("Handling cached response...", "url", req.URL)

	if clientHd.Range.IsSome() {
		if err := p.handleRangeRequest(r, req, cached, key, clientHd); err != nil {
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

	r.SetHeaders(cached.Metadata.Object.Header)
	r.SetHeader("Accept-Ranges", "bytes")
	r.SetHeader("ETag", cached.Metadata.Object.ETag)
	r.SetHeader("Last-Modified", cached.Metadata.Object.LastModified.Format(http.TimeFormat))

	slog.Info("Serving cached response", "url", req.URL, "key", key)
	return sendResponse(r, cached.Data, http.StatusOK, req)
}

func (p *cachingMitmProxyHandler) processHTTPRequest(r responder.Responder, req *http.Request) error {
	slog.Info("Processing HTTP request", "remote_addr", req.RemoteAddr, "method", req.Method, "url", req.URL)

	var upstreamDefaultHttps bool
	cfgLock := config.Global.Immutable()
	cfgLock.Read(func(c *config.Config) {
		upstreamDefaultHttps = c.UpstreamDefaultHttps.Read()
	})

	clientHd := headers.ParseHeaderDirective(req.Header)
	// Remove clients conditionals so we don't forward them to upstream.
	clientHd.ConditionalHeaders.StripFromHeader(req.Header)

	key := cache.MakeFromRequest(req)
	cached, err := p.cache.Get(key)
	if err != nil && !errors.Is(err, cache.ErrCacheMiss) {
		slog.Error("Error getting cache for key", "key", key, "error", err)
		r.WriteError("error retrieving from cache", http.StatusInternalServerError)
		return fmt.Errorf("%w: %v", ErrCacheGetFailed, err)
	}

	if cached != nil {
		defer cached.Data.Close() // Ensure we close the cached data when done

		if !cached.Stale {

			if err := p.handleCachedResponse(r, req, cached, key, clientHd); err != nil {
				slog.Error("Error handling cached response", "url", req.URL, "key", key, "error", err)
				return err
			}
			return nil
		}

		slog.Info("Cached response is stale", "url", req.URL, "key", key)

		// Cache is stale: set conditional headers if available
		if cached.Metadata.Object.ETag != "" {
			req.Header.Set("If-None-Match", cached.Metadata.Object.ETag)
		}
		if !cached.Metadata.Object.LastModified.IsZero() {
			req.Header.Set("If-Modified-Since", cached.Metadata.Object.LastModified.Format(http.TimeFormat))
		}
	}

	slog.Info("Sending request to upstream", "url", req.URL)
	resp, err := sendRequestToTarget(req, upstreamDefaultHttps)
	if err != nil {
		slog.Error("Error sending request to upstream target", "url", req.URL, "error", err)
		r.WriteError("error sending request to upstream target", http.StatusBadGateway)
		return fmt.Errorf("%w: %v", ErrBadGateway, err)
	}
	defer resp.Body.Close() // Ensure we close the response body when done

	if err := p.handleUpstreamResponse(r, req, resp, cached, key, clientHd); err != nil {
		slog.Error("Error handling upstream response", "url", req.URL, "key", key, "error", err)
	}
	return nil
}

func (p *cachingMitmProxyHandler) handleHTTP(w http.ResponseWriter, proxyReq *http.Request) error {
	slog.Info("Handling HTTP request", "host", proxyReq.Host, "remote_addr", proxyReq.RemoteAddr)

	metrics.Global.Requests.HTTPProxyRequests.Increment()

	r := responder.NewHTTPResponder(w)
	return p.processHTTPRequest(r, proxyReq)
}

// Helper function to hijack the connection from the ResponseWriter.
// NOTE: Remember to close the hijacked connection when done.
func hijackConnection(w http.ResponseWriter) (net.Conn, error) {
	// "Hijack" the client connection to get a TCP (or TLS) socket we can read and write arbitrary data to/from.
	hj, ok := w.(http.Hijacker)
	if !ok {
		slog.Error("Could not hijack connection for provided ResponseWriter", "host", w.Header().Get("Host"), "writer_type", fmt.Sprintf("%T", w))
		return nil, ErrHijackNotSupported
	}

	// Hijack the connection to get the underlying net.Conn.
	clientConn, _, err := hj.Hijack()
	if err != nil {
		slog.Error("Failed to hijack connection", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrHijackFailed, err)
	}

	return clientConn, nil
}

func (p *cachingMitmProxyHandler) handleCONNECT(w http.ResponseWriter, proxyReq *http.Request) error {
	slog.Info("Handling CONNECT request", "url", proxyReq.URL, "remote_addr", proxyReq.RemoteAddr)

	metrics.Global.Requests.HTTPSProxyRequests.Increment()

	clientConn, err := hijackConnection(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	defer clientConn.Close() // Ensure we always close the hijacked connection

	intermediateResponder := responder.NewRawHTTPResponder(clientConn)

	tlsCert, err := p.ca.GetCertForHost(proxyReq.Host)
	if err != nil {
		// can't use http.Error after hijacking, so we write directly
		slog.Error("Error getting TLS certificate", "host", proxyReq.Host, "error", err)
		intermediateResponder.WriteError("Error getting TLS certificate", http.StatusInternalServerError)
		return fmt.Errorf("%w: %v", ErrTLSCertFailed, err)
	}

	// Send an HTTP OK response back to the client. This initiates the CONNECT
	// tunnel. From this point on the client will assume it's connected directly
	// to the target.
	if err := intermediateResponder.WriteEmpty(http.StatusOK); err != nil {
		slog.Error("Failed to write HTTP OK response to client", "error", err)
		return fmt.Errorf("%w: %v", ErrClientResponseFailed, err)
	}
	slog.Debug("Sent HTTP 200 OK response to client, established CONNECT tunnel")

	// Configure a new TLS server, pointing it at the client connection, using
	// our certificate. This server will now pretend being the target.
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{*tlsCert},
	}
	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()

	// Create a buffered reader for the client connection. This is required to
	// use http package functions with this connection.
	connReader := bufio.NewReader(tlsConn)
	responder := responder.NewRawHTTPResponder(tlsConn)

	slog.Debug("Entering request loop for CONNECT tunnel")
	for {
		// Read next HTTP request from client.
		req, err := http.ReadRequest(connReader)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			slog.Error("Error reading request from client", "remote_addr", proxyReq.RemoteAddr, "error", err)
			return fmt.Errorf("%w: %v ", ErrReadRequestFailed, err)
		}

		if err := p.processHTTPRequest(responder, req); err != nil {
			slog.Error("Error processing HTTP request", "remote_addr", proxyReq.RemoteAddr, "error", err)
		}
	}

	return nil
}
