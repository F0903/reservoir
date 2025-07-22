package proxy

import (
	"apt_cacher_go/cache"
	"apt_cacher_go/config"
	"apt_cacher_go/metrics"
	"apt_cacher_go/proxy/certs"
	"apt_cacher_go/proxy/responder"
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
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
	return &cachingMitmProxyHandler{
		ca:    ca,
		cache: cache.NewFileCache[cachedRequestInfo](cacheDir, config.Global.CacheCleanupInterval.Cast(), ctx),
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
	if config.Global.AlwaysCache {
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

	clientDirective := parseCacheDirective(req.Header)

	// The way we handle handle caching should already line up with the client's expectations, so we can remove these headers.
	// If we don't remove them, we might end up getting an unexpected response from the upstream server.
	clientDirective.conditionalHeaders.removeFromHeader(req.Header)

	// Remove headers that we don't support before anything else.
	// Otherwise we end up sending headers and getting responses that we don't know how to handle.
	removeUnsupportedHeaders(req.Header)

	key := cache.MakeFromRequest(req)

	cached, err := p.cache.Get(key)
	if err != nil && !errors.Is(err, cache.ErrorCacheMiss) {
		err := fmt.Errorf("error getting cache for key %v: %v", key, err)
		r.Error("error retrieving from cache", http.StatusInternalServerError)
		return err
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
	resp, err := sendRequestToTarget(req, config.Global.UpstreamDefaultHttps)
	if err != nil {
		slog.Error("Error sending request to upstream target", "error", err)
		err := fmt.Errorf("error sending request to target (%v): %v", req.URL, err)
		r.Error("error sending request to upstream target", http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close() // Ensure we close the response body when done

	if resp.StatusCode == http.StatusNotModified {
		if cached == nil {
			err := fmt.Errorf("received 304 Not Modified but no cached response found for '%v' with key '%v'\nRequest headers might be malformed.\nRequest headers: %v", req.URL, key, req.Header)
			r.Error("malformed state", http.StatusInternalServerError)
			return err
		}

		err := p.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[cachedRequestInfo]) {
			// Update the metadata to reflect that the cached response is still valid.
			maxAge := config.Global.DefaultCacheMaxAge.Cast()
			meta.Expires = time.Now().Add(maxAge)
		})
		if err != nil {
			err := fmt.Errorf("error updating cache metadata for '%v' with key '%v': %v", req.URL, key, err)
			r.Error("error updating cache metadata", http.StatusInternalServerError)
			return err
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
			err := fmt.Errorf("error caching response for '%v' with key '%v': %v", req.URL, key, err)
			r.Error("error caching response", http.StatusInternalServerError)
			return err
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
		err := fmt.Errorf("hijacking not supported for target host. Hijacking only works with servers that support HTTP 1.x")
		return nil, err
	}

	// Hijack the connection to get the underlying net.Conn.
	clientConn, _, err := hj.Hijack()
	if err != nil {
		err := fmt.Errorf("hijack failed: %v", err)
		return nil, err
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
		intermediateResponder.Error("Error getting TLS certificate", http.StatusInternalServerError)
		return err
	}

	// Send an HTTP OK response back to the client; this initiates the CONNECT
	// tunnel. From this point on the client will assume it's connected directly
	// to the target.
	if err := intermediateResponder.WriteEmpty(http.StatusOK); err != nil {
		return fmt.Errorf("failed to write HTTP OK response to client: %v", err)
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
			return fmt.Errorf("error reading request from client (%v): %w", proxyReq.RemoteAddr, err)
		}

		p.processHTTPRequest(responder, req)
	}

	return nil
}
