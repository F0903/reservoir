package proxy

import (
	"apt_cacher_go/cache"
	"apt_cacher_go/config"
	"apt_cacher_go/proxy/certs"
	"apt_cacher_go/proxy/responder"
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
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
	cache cache.Cache[cachedRequestInfo]
}

func newCachingMitmProxyHandler(cacheDir string, ca certs.CertAuthority) (*cachingMitmProxyHandler, error) {
	return &cachingMitmProxyHandler{
		ca:    ca,
		cache: cache.NewFileCache[cachedRequestInfo](cacheDir),
	}, nil
}

func (p *cachingMitmProxyHandler) ServeHTTP(w http.ResponseWriter, proxyReq *http.Request) {
	if proxyReq.Method == http.MethodConnect {
		if err := p.handleCONNECT(w, proxyReq); err != nil {
			log.Printf("Error handling CONNECT request: %v", err)
			return
		}
	} else {
		if err := p.handleHTTP(w, proxyReq); err != nil {
			log.Printf("Error handling HTTP request: %v", err)
			return
		}
	}
}

func (p *cachingMitmProxyHandler) getCached(key *cache.CacheKey, req *http.Request) (*cache.Entry[cachedRequestInfo], error) {
	cached, err := p.cache.Get(key)
	if errors.Is(err, cache.ErrorCacheMiss) {
		log.Printf("Cache miss for key %v", key)
		return nil, nil // Cache miss, return nil to indicate no cached entry
	} else if cached == nil && !errors.Is(err, cache.ErrorCacheMiss) {
		return nil, fmt.Errorf("error retrieving from cache for key %v: %w", key, err)
	}

	if cached.Stale {
		log.Printf("Cached response for %v is stale. Setting conditional headers...", req.Host)

		// Cache is stale: set conditional headers
		if cached.Metadata.Object.ETag != "" {
			req.Header.Set("If-None-Match", cached.Metadata.Object.ETag)
		}
		if !cached.Metadata.Object.LastModified.IsZero() {
			req.Header.Set("If-Modified-Since", cached.Metadata.Object.LastModified.Format(http.TimeFormat))
		}

		return cached, nil
	}

	return cached, nil
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
	if err := r.Write(http.StatusOK, body); err != nil {
		log.Printf("error writing response for '%v': %v", req.URL, err)
	}
}

func (p *cachingMitmProxyHandler) processHTTPRequest(r responder.Responder, req *http.Request) error {
	log.Printf("Processing HTTP request %s -> %s %s", req.RemoteAddr, req.Method, req.URL)

	clientDirective := parseCacheDirective(req.Header)

	// The way we handle handle caching should already line up with the client's expectations, so we can remove these headers.
	// If we don't remove them, we might end up getting an unexpected response from the upstream server.
	clientDirective.conditionalHeaders.removeFromHeader(req.Header)

	// Remove headers that we don't support before anything else.
	// Otherwise we end up sending headers and getting responses that we don't know how to handle.
	removeUnsupportedHeaders(req.Header)

	key := cache.MakeFromRequest(req)

	cached, err := p.getCached(key, req)
	if err != nil {
		err := fmt.Errorf("error getting cache for key %v: %v", key, err)
		r.Error("error retrieving from cache", http.StatusInternalServerError)
		return err
	}

	if cached != nil {
		defer cached.Data.Close() // Ensure we close the cached data when done
		if !cached.Stale {
			log.Printf("Serving cached response for '%v' with key '%v'", req.URL, key)
			sendResponse(r, cached.Data, cached.Metadata.Object.Header, req)
			return nil
		}
	}

	log.Printf("Sending request to upstream '%v'", req.URL)
	resp, err := sendRequestToTarget(req, config.Global.UpstreamDefaultHttps)
	if err != nil {
		log.Printf("error sending request to target (%v): %v", req.URL, err)
		r.Error("error sending request to upstream target", http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close() // Ensure we close the response body when done

	if resp.StatusCode == http.StatusNotModified {
		if cached == nil {
			log.Printf("Received 304 Not Modified but no cached response found for '%v' with key '%v'\nRequest headers might be malformed.\nRequest headers: %v", req.URL, key, req.Header)
			err := fmt.Errorf("received 304 Not Modified but no cached response found for '%v' with key '%v'", req.URL, key)
			r.Error("malformed state", http.StatusInternalServerError)
			return err
		}

		err := p.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[cachedRequestInfo]) {
			// Update the metadata to reflect that the cached response is still valid.
			maxAge := config.Global.DefaultCacheMaxAge.Cast()
			meta.Expires = time.Now().Add(maxAge)
		})
		if err != nil {
			errMsg := fmt.Errorf("error updating cache metadata for '%v' with key '%v': %v", req.URL, key, err)
			log.Printf("error updating cache metadata for '%v' with key '%v': %v", req.URL, key, err)
			r.Error("error updating cache metadata", http.StatusInternalServerError)
			return errMsg
		}

		log.Printf("Origin server returned 304 Not Modified, serving cached response for '%v' with key '%v'", req.URL, key)
		sendResponse(r, cached.Data, cached.Metadata.Object.Header, req)
		return nil
	}

	var data io.Reader = resp.Body

	upstreamDirective := parseCacheDirective(resp.Header)

	if shouldResponseBeCached(resp, upstreamDirective) {
		log.Printf("Caching response for %s '%v' with key '%v'", resp.Status, req.URL, key)

		lastModified := time.Now()
		if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
			lastModified = t
		}

		etag := resp.Header.Get("ETag")

		maxAge := upstreamDirective.getExpiresOrDefault(config.Global.DefaultCacheMaxAge.Cast())
		entry, err := p.cache.Cache(key, resp.Body, maxAge, cachedRequestInfo{
			ETag:         etag,
			LastModified: lastModified,
			Header:       resp.Header,
		})
		if err != nil {
			log.Printf("error caching response for '%v' with key '%v': %v", req.URL, key, err)
			r.Error("error caching response", http.StatusInternalServerError)
			return fmt.Errorf("error caching response for '%v' with key '%v': %v", req.URL, key, err)
		}
		defer entry.Data.Close() // Ensure we close the cached data when done

		data = entry.Data
	}

	log.Printf("Sending response for '%v' with status %d", req.URL, resp.StatusCode)
	sendResponse(r, data, resp.Header, req)
	return nil
}

func (p *cachingMitmProxyHandler) handleHTTP(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("HTTP request to %v (from %v)", proxyReq.Host, proxyReq.RemoteAddr)

	responder := responder.NewHTTPResponder(w)
	return p.processHTTPRequest(responder, proxyReq)
}

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
	log.Printf("CONNECT request to %v (from %v)", proxyReq.URL, proxyReq.RemoteAddr)

	clientConn, err := hijackConnection(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	defer clientConn.Close() // Ensure we always close the hijacked connection

	tlsCert, err := p.ca.GetCertForHost(proxyReq.Host)
	if err != nil {
		// Note: can't use http.Error after hijacking, so we write directly
		clientConn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
		return err
	}

	// Send an HTTP OK response back to the client; this initiates the CONNECT
	// tunnel. From this point on the client will assume it's connected directly
	// to the target.
	if _, err := clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		return fmt.Errorf("failed to write HTTP OK response to client: %v", err)
	}
	log.Print("Sent HTTP 200 OK response to client, established CONNECT tunnel")

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

	log.Print("Entering request loop for CONNECT tunnel")
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
