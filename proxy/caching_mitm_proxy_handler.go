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
	"reservoir/proxy/responder"
	"time"
)

var (
	ErrCacheGetFailed       = errors.New("error getting cache for key")
	ErrSendRequestUpstream  = errors.New("error sending request to upstream target")
	ErrNoCachedResponse304  = errors.New("received 304 Not Modified but no cached response found")
	ErrUpdateCacheMetadata  = errors.New("error updating cache metadata")
	ErrCacheResponseFailed  = errors.New("error caching response")
	ErrHijackNotSupported   = errors.New("hijacking not supported for target host")
	ErrHijackFailed         = errors.New("hijack failed")
	ErrTLSCertFailed        = errors.New("error getting TLS certificate")
	ErrClientResponseFailed = errors.New("failed to write HTTP OK response to client")
	ErrReadRequestFailed    = errors.New("error reading request from client")
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

func shouldResponseBeCached(resp *http.Response, upstreamDirective *cacheDirective) bool {
	cfgLock := config.Global.Immutable()

	var alwaysCache bool
	cfgLock.Read(func(c *config.Config) {
		alwaysCache = c.AlwaysCache.Read()
	})

	if alwaysCache {
		return true
	}

	return upstreamDirective.shouldCache() &&
		resp.StatusCode == http.StatusOK &&
		(resp.Request.Method == http.MethodGet ||
			resp.Request.Method == http.MethodHead)
}

func sendResponse(r responder.Responder, resp io.Reader, header http.Header, req *http.Request) {
	body := resp
	if req.Method == http.MethodHead {
		body = http.NoBody
	}

	r.SetHeader(header)
	written, err := r.Write(http.StatusOK, body)
	if err != nil {
		slog.Error("Error writing response", "url", req.URL, "error", err)
	}

	metrics.Global.Requests.BytesServed.Add(written)
	slog.Info("Response sent", "url", req.URL, "bytes_written", written)
}

func (p *cachingMitmProxyHandler) processHTTPRequest(r responder.Responder, req *http.Request) error {
	slog.Info("Processing HTTP request", "remote_addr", req.RemoteAddr, "method", req.Method, "url", req.URL)

	cfgLock := config.Global.Immutable()

	var defaultCacheMaxAge time.Duration
	var upstreamDefaultHttps bool
	cfgLock.Read(func(c *config.Config) {
		defaultCacheMaxAge = c.DefaultCacheMaxAge.Read().Cast()
		upstreamDefaultHttps = c.UpstreamDefaultHttps.Read()
	})

	clientDirective := parseCacheDirective(req.Header)
	// The way we handle handle caching should already line up with the client's expectations, so we can remove these headers.
	// If we don't remove them, we might end up getting an unexpected response from the upstream server.
	clientDirective.conditionalHeaders.removeFromHeader(req.Header)

	// Remove headers that we don't support before anything else.
	// Otherwise we end up sending headers and getting responses that we don't know how to handle.
	removeUnsupportedHeaders(req.Header)

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
			slog.Info("Serving cached response", "url", req.URL, "key", key)
			sendResponse(r, cached.Data, cached.Metadata.Object.Header, req)
			return nil
		}

		slog.Warn("Cached response is stale", "url", req.URL, "key", key)

		// Cache is stale: set conditional headers
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
		return fmt.Errorf("%w: %v", ErrSendRequestUpstream, err)
	}
	defer resp.Body.Close() // Ensure we close the response body when done

	if resp.StatusCode == http.StatusNotModified {
		if cached == nil {
			slog.Error("Received 304 Not Modified but no cached response found", "url", req.URL, "key", key, "headers", req.Header)
			r.WriteError("malformed state", http.StatusInternalServerError)
			return ErrNoCachedResponse304
		}

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
		sendResponse(r, cached.Data, cached.Metadata.Object.Header, req)
		return nil
	}

	var data io.Reader = resp.Body

	upstreamDirective := parseCacheDirective(resp.Header)

	if shouldResponseBeCached(resp, upstreamDirective) {
		slog.Info("Caching response", "status", resp.Status, "url", req.URL, "key", key)

		lastModified := time.Now()
		if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
			lastModified = t
		}

		etag := resp.Header.Get("ETag")

		maxAge := upstreamDirective.getExpiresOrDefault()
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

	slog.Info("Sending response", "url", req.URL, "status", resp.StatusCode)
	sendResponse(r, data, resp.Header, req)
	return nil
}

func (p *cachingMitmProxyHandler) handleHTTP(w http.ResponseWriter, proxyReq *http.Request) error {
	slog.Info("Handling HTTP request", "host", proxyReq.Host, "remote_addr", proxyReq.RemoteAddr)

	metrics.Global.Requests.HTTPProxyRequests.Increment()

	responder := responder.NewHTTPResponder(w)
	return p.processHTTPRequest(responder, proxyReq)
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

	// Send an HTTP OK response back to the client; this initiates the CONNECT
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

	// Create a buffered reader for the client connection; this is required to
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

		p.processHTTPRequest(responder, req)
	}

	return nil
}
