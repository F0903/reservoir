package proxy

import (
	"apt_cacher_go/cache"
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

type CachingMitmProxy struct {
	ca            certs.CertAuthority
	cache         cache.Cache[cachedRequestInfo]
	defaultMaxAge time.Duration
}

// createMitmProxy creates a new MITM proxy. It should be passed the filenames
// for the certificate and private key of a certificate authority trusted by the
// client's machine.
func NewCachingMitmProxy(cacheDir string, ca certs.CertAuthority) (*CachingMitmProxy, error) {
	return &CachingMitmProxy{
		ca:            ca,
		cache:         cache.NewFileCache[cachedRequestInfo](cacheDir),
		defaultMaxAge: 1 * time.Hour, // Default expiration time for cached responses
	}, nil
}

func (p *CachingMitmProxy) ServeHTTP(w http.ResponseWriter, proxyReq *http.Request) {
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

func (p *CachingMitmProxy) getCached(key *cache.CacheKey, req *http.Request) (*cache.Entry[cachedRequestInfo], error) {
	cached, err := p.cache.Get(key)
	if errors.Is(err, cache.ErrorCacheMiss) {
		log.Printf("Cache miss for key %v", key)
		return nil, nil // Cache miss, return nil to indicate no cached entry
	} else if cached == nil && !errors.Is(err, cache.ErrorCacheMiss) {
		return nil, fmt.Errorf("error retrieving from cache for key %v: %w", key, err)
	}

	// If the cached response is still valid, serve it directly.
	if cached.Stale {
		log.Printf("Cached response for %v is stale. Setting conditional headers...", req.Host)

		// Cache is stale: set conditional headers
		if cached.Metadata.Object.ETag != "" {
			req.Header.Set("If-None-Match", cached.Metadata.Object.ETag)
		}
		if !cached.Metadata.Object.LastModified.IsZero() {
			req.Header.Set("If-Modified-Since", cached.Metadata.Object.LastModified.Format(http.TimeFormat))
		}

		return nil, nil // Return nil to indicate that we need to revalidate the request
	}

	return cached, nil
}

func (p *CachingMitmProxy) processHEAD(r responder.Responder, req *http.Request, key *cache.CacheKey) error {
	log.Printf("Processing HEAD request...")

	cached, err := p.getCached(key, req)
	if err != nil {
		err := fmt.Errorf("error getting cache for key %v: %v", key, err)
		r.Error(err, http.StatusInternalServerError)
		return err
	} else if cached != nil {
		defer cached.Data.Close() // Close the cached data stream when we return

		log.Printf("Serving cached response for %v", req.Host)
		r.SetHeader(cached.Metadata.Object.Header)
		if err := r.WriteEmpty(http.StatusOK); err != nil {
			log.Printf("error writing cached response for key %v: %v", key, err)
		}
		log.Printf("Cached response for %v is stale. Revalidating...", req.Host)
	}

	// Change request URL to point to the target server.
	changeRequestToTarget(req)
	// Remove hop-by-hop headers in the request that should not be forwarded to the target server.
	removeHopByHopHeaders(req.Header)

	log.Printf("Making HEAD request to %v", req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to target (%v): %w", req.URL, err)
	}
	defer resp.Body.Close()

	// If 304 Not Modified and we have a cached response, serve the cached response.
	if resp.StatusCode == http.StatusNotModified && cached != nil {
		log.Printf("Origin server returned 304 Not Modified, serving cached response for %v", req.URL)
		p.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[cachedRequestInfo]) {
			// Update the metadata to reflect that the cached response is still valid.
			meta.Expires = time.Now().Add(p.defaultMaxAge)
		})
		r.SetHeader(cached.Metadata.Object.Header)
		if err := r.WriteEmpty(http.StatusOK); err != nil {
			log.Printf("error writing cached response for key %v: %v", key, err)
		}
		return nil
	}

	directive := parseCacheDirective(resp.Header)

	// Only cache if the status code is OK and caching is not disabled.
	// It is important to make sure only 200 OK responses are cached to
	// avoid mistakenly writing empty responses among other things.
	if directive.shouldCache() && resp.StatusCode == http.StatusOK {
		log.Printf("Caching response for %v", req.URL)

		lastModified := time.Now()
		if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
			lastModified = t
		}

		etag := resp.Header.Get("ETag")

		entry, err := p.cache.Cache(key, resp.Body, directive.getExpiresOrDefault(p.defaultMaxAge), cachedRequestInfo{
			ETag:         etag,
			LastModified: lastModified,
			Header:       resp.Header,
		})
		if err != nil {
			log.Printf("error caching response for %v: %v", req.URL, err)
			r.Error(err, http.StatusInternalServerError)
			return fmt.Errorf("error caching response for %v: %v", req.URL, err)
		}
		entry.Data.Close() // Since we are not writing the body, we can close it immediately.
	}

	// Send the target server's response headers back to the client.
	r.SetHeader(resp.Header)
	if err := r.WriteEmpty(resp.StatusCode); err != nil {
		log.Printf("error writing response back to client (%v): %v\n", req.RemoteAddr, err)
	}
	return nil
}

func (p *CachingMitmProxy) processGET(r responder.Responder, req *http.Request, key *cache.CacheKey) error {
	log.Printf("Processing GET request...")

	cached, err := p.getCached(key, req)
	if err != nil {
		err := fmt.Errorf("error getting cache for key %v: %v", key, err)
		r.Error(err, http.StatusInternalServerError)
		return err
	} else if cached != nil {
		log.Printf("Serving cached response for %v", req.Host)
		r.SetHeader(cached.Metadata.Object.Header)
		if err := r.Write(http.StatusOK, cached.Data); err != nil {
			log.Printf("error writing cached response for key %v: %v", key, err)
		}
		log.Printf("Cached response for %v is stale. Revalidating...", req.Host)
	}

	resp, err := sendRequestToTarget(req)
	if err != nil {
		log.Printf("error sending request to target (%v): %v", req.URL, err)
		r.Error(err, http.StatusBadGateway)
		return err
	}

	// If 304 Not Modified and we have a cached response, serve the cached response.
	if resp.StatusCode == http.StatusNotModified && cached != nil {
		log.Printf("Origin server returned 304 Not Modified, serving cached response for %v", req.URL)
		p.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[cachedRequestInfo]) {
			// Update the metadata to reflect that the cached response is still valid.
			meta.Expires = time.Now().Add(p.defaultMaxAge)
		})
		r.SetHeader(cached.Metadata.Object.Header)
		if err := r.Write(http.StatusOK, cached.Data); err != nil {
			log.Printf("error writing cached response for key %v: %v", key, err)
		}
		return nil
	}

	var data io.ReadCloser = resp.Body

	upstreamDirective := parseCacheDirective(resp.Header)

	// Only cache if the status code is OK and caching is not disabled.
	// It is important to make sure only 200 OK responses are cached to
	// avoid mistakenly writing empty responses among other things.
	if upstreamDirective.shouldCache() && resp.StatusCode == http.StatusOK {
		log.Printf("Caching response for %v", req.URL)

		lastModified := time.Now()
		if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
			lastModified = t
		}

		etag := resp.Header.Get("ETag")

		entry, err := p.cache.Cache(key, resp.Body, upstreamDirective.getExpiresOrDefault(p.defaultMaxAge), cachedRequestInfo{
			ETag:         etag,
			LastModified: lastModified,
			Header:       resp.Header,
		})
		if err != nil {
			log.Printf("error caching response for %v: %v", req.URL, err)
			r.Error(err, http.StatusInternalServerError)
			return fmt.Errorf("error caching response for %v: %v", req.URL, err)
		}

		data.Close() // Close the old data stream since we are now using the cached entry.
		data = entry.Data
	}

	// Send the target server's response back to the client.
	r.SetHeader(resp.Header)
	if err := r.Write(resp.StatusCode, data); err != nil {
		log.Printf("error writing response back to client (%v): %v\n", req.RemoteAddr, err)
	}
	return nil
}

func (p *CachingMitmProxy) processHTTPRequest(r responder.Responder, req *http.Request, key *cache.CacheKey) error {
	log.Printf("Processing HTTP request %s -> %s %s", req.RemoteAddr, req.Method, req.URL)

	// Remove headers that we don't support before anything else.
	// Otherwise we end up sending headers and getting responses that we don't know how to handle.
	removeUnsupportedHeaders(req.Header)

	switch req.Method {
	case http.MethodGet:
		return p.processGET(r, req, key)
	case http.MethodHead:
		return p.processHEAD(r, req, key)
	// Add more methods as needed (e.g., POST, PUT, DELETE)
	default:
		err := fmt.Errorf("unsupported HTTP method: %s", req.Method)
		r.Error(err, http.StatusMethodNotAllowed)
		return err
	}
}

func (p *CachingMitmProxy) handleHTTP(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("HTTP request to %v (from %v)", proxyReq.Host, proxyReq.RemoteAddr)

	key := cache.MakeFromRequest(proxyReq)

	responder := responder.NewHTTPResponder(w)
	return p.processHTTPRequest(responder, proxyReq, key)
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

func (p *CachingMitmProxy) handleCONNECT(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("CONNECT request to %v (from %v)", proxyReq.URL, proxyReq.RemoteAddr)

	key := cache.MakeFromRequest(proxyReq)

	clientConn, err := hijackConnection(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	tlsCert, err := p.ca.GetCertForHost(proxyReq.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		Certificates: []tls.Certificate{tlsCert},
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

		p.processHTTPRequest(responder, req, key)
	}

	return nil
}
