package proxy

import (
	"apt_cacher_go/cache"
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type cacheControl struct {
	noCache        bool
	mustRevalidate bool
	maxAge         time.Duration
}

type requestCacheStatus struct {
	noCache bool
	expires time.Time
}

type cachedRequestInfo struct {
	ETag         string
	LastModified time.Time
	Header       http.Header
}

type CachingMitmProxy struct {
	caCert         *x509.Certificate
	caKey          any
	cache          cache.Cache[cachedRequestInfo]
	defaultExpires time.Duration
}

// createMitmProxy creates a new MITM proxy. It should be passed the filenames
// for the certificate and private key of a certificate authority trusted by the
// client's machine.
func NewCachingMitmProxy(caCertFile, caKeyFile string, cacheDir string) (*CachingMitmProxy, error) {
	caCert, caKey, err := loadX509KeyPair(caCertFile, caKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate and key: %v", err)
	}
	log.Printf("Loaded CA certificate: '%v' (IsCA=%v)\n", caCert.Subject.CommonName, caCert.IsCA)

	return &CachingMitmProxy{
		caCert:         caCert,
		caKey:          caKey,
		cache:          cache.NewFileCache[cachedRequestInfo](cacheDir),
		defaultExpires: 1 * time.Hour, // Default expiration time for cached responses
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

func (p *CachingMitmProxy) parseCacheControl(header http.Header) (*cacheControl, error) {
	ccHeader := header.Get("Cache-Control")
	if ccHeader == "" {
		return nil, nil // No Cache-Control header, nothing to parse
	}

	cc := &cacheControl{}
	// Parse the Cache-Control header for max-age directive
	for directive := range strings.SplitSeq(ccHeader, ",") {
		directive = strings.TrimSpace(directive)
		if directive == "no-cache" || directive == "no-store" {
			cc.noCache = true
		} else if directive == "must-revalidate" || directive == "proxy-revalidate" {
			// must-revalidate is used to indicate that the cache must revalidate
			// the response with the origin server before serving it to the client.
			// This is useful for ensuring that the client always gets the most up-to-date response.
			// proxy-revalidate is similar, but specifically for proxies. (we interpret it the same way)
			cc.mustRevalidate = true
		} else if strings.HasPrefix(directive, "max-age=") {
			// max-age directive specifies the maximum amount of time a response is considered fresh in seconds.
			maxAgeStr := strings.TrimPrefix(directive, "max-age=")
			maxAge, err := strconv.ParseInt(maxAgeStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse max-age: %v", err)
			}
			cc.maxAge = time.Duration(maxAge) * time.Second
		}
	}

	return cc, nil
}

func (p *CachingMitmProxy) parseRequestCacheStatus(header http.Header) requestCacheStatus {
	// For now we only use max-age from Cache-Control
	cc, err := p.parseCacheControl(header)
	if err == nil {
		return requestCacheStatus{
			noCache: cc.noCache,
			expires: time.Now().Add(cc.maxAge),
		}
	}
	log.Printf("Error parsing Cache-Control header: %v", err)

	expires, err := http.ParseTime(header.Get("Expires"))
	if err == nil {
		return requestCacheStatus{
			noCache: false,
			expires: expires,
		}
	}
	log.Printf("Error parsing Expires header: %v", err)

	defaultExpires := time.Now().Add(p.defaultExpires)
	log.Printf("Using default expiration time of %v", defaultExpires)
	return requestCacheStatus{
		noCache: false,
		expires: defaultExpires,
	}
}

// Checks if the request is already cached. If it is, it returns the cached entry.
// If the entry is expired, it removes it from the cache and returns nil.
func (p *CachingMitmProxy) getCached(key *cache.CacheKey) (*cache.Entry[cachedRequestInfo], error) {
	if cached, err := p.cache.Get(key); err == nil {
		log.Printf("Cache hit for %v", key)

		if cached.Metadata.Expires.Before(cached.Metadata.TimeWritten) {
			log.Printf("Cache entry for %v is expired, removing from cache", key)
			p.cache.Delete(key)
		} else {
			log.Printf("Cache entry for %v is valid, serving from cache", key)
			return cached, nil
		}

	} else if !errors.Is(err, cache.ErrorCacheMiss) {
		return nil, fmt.Errorf("error retrieving from cache for %v: %w", key, err)
	}

	log.Printf("Cache miss for %v", key)
	return nil, nil
}

func makeHTTPResponseWithStream(status int, stream io.ReadCloser, header http.Header) *http.Response {
	return &http.Response{
		StatusCode: status,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Body:       stream,
	}
}

func makeHTTPResponseWithString(status int, text string, header http.Header) *http.Response {
	return makeHTTPResponseWithStream(status, io.NopCloser(strings.NewReader(text)), header)
}

func (p *CachingMitmProxy) processHTTPRequest(w io.Writer, req *http.Request, key *cache.CacheKey) error {
	log.Printf("Received HTTP request from client (%v): %s %s", req.RemoteAddr, req.Method, req.URL)

	cached, err := p.getCached(key)
	if err != nil {
		err := fmt.Errorf("error getting cache for %v: %v", req.Host, err)
		makeHTTPResponseWithString(http.StatusInternalServerError, "Error retrieving cached response: "+err.Error(), make(http.Header)).Write(w)
		return err
	}

	if cached != nil {
		defer cached.Data.Close() // Close the cached data stream when we return

		// If the cached response is still valid, serve it directly.
		if time.Now().Before(cached.Metadata.Expires) {
			log.Printf("Serving cached response for %v", req.Host)
			makeHTTPResponseWithStream(http.StatusOK, cached.Data, cached.Metadata.Object.Header).Write(w)
			return nil
		}
		log.Printf("Cached response for %v is stale. Revalidating...", req.Host)

		// Cache is stale: set conditional headers
		if cached.Metadata.Object.ETag != "" {
			req.Header.Set("If-None-Match", cached.Metadata.Object.ETag)
		}
		if !cached.Metadata.Object.LastModified.IsZero() {
			req.Header.Set("If-Modified-Since", cached.Metadata.Object.LastModified.UTC().Format(http.TimeFormat))
		}
	}

	// Take the request and changes its destination to be forwarded to the target server.
	// The target server is specified in the original CONNECT request, which is proxyReq.
	changeRequestToTarget(req, req.Host)
	// Remove hop-by-hop headers in the request that should not be forwarded to the target server.
	removeHopByHopHeaders(req.Header)

	log.Printf("Making request to %v", req.URL)
	// Send the request to the target server and log the response.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to target (%v): %w", req.URL, err)
	}
	defer resp.Body.Close()
	log.Printf("sent request to target %v, got response status: %s", req.URL, resp.Status)

	// If 304 Not Modified, serve cached response
	if resp.StatusCode == http.StatusNotModified && cached != nil {
		log.Printf("Origin server returned 304 Not Modified, serving cached response for %v", req.URL)
		makeHTTPResponseWithStream(http.StatusOK, cached.Data, cached.Metadata.Object.Header).Write(w)
		p.cache.UpdateMetadata(key, func(meta *cache.EntryMetadata[cachedRequestInfo]) {
			// Update the metadata to reflect that the cached response is still valid.
			meta.Expires = time.Now().Add(p.defaultExpires)
		})
		return nil
	}

	// Remove any hop-by-hop headers in the response that should not be forwarded to the client.
	removeHopByHopHeaders(resp.Header)

	lastModified := time.Now()
	if t, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
		lastModified = t
	}
	etag := resp.Header.Get("ETag")
	requestCacheStatus := p.parseRequestCacheStatus(resp.Header)

	var data io.Reader = resp.Body

	if !requestCacheStatus.noCache {
		entry, err := p.cache.Cache(key, resp.Body, requestCacheStatus.expires, cachedRequestInfo{
			ETag:         etag,
			LastModified: lastModified,
			Header:       resp.Header,
		})
		if err != nil {
			log.Printf("error caching response for %v: %v", req.URL, err)
			makeHTTPResponseWithString(http.StatusInternalServerError, "Error caching response: "+err.Error(), make(http.Header)).Write(w)
			return fmt.Errorf("error caching response for %v: %v", req.URL, err)
		}
		defer entry.Data.Close() // Ensure we close the cached data stream

		data = entry.Data
	}

	// Send the target server's response back to the client.
	if err := makeHTTPResponseWithStream(resp.StatusCode, io.NopCloser(data), resp.Header).Write(w); err != nil {
		log.Printf("error writing response back to client (%v): %v\n", req.RemoteAddr, err)
	}
	return nil
}

func (p *CachingMitmProxy) handleHTTP(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("HTTP request to %v (from %v)", proxyReq.Host, proxyReq.RemoteAddr)

	key := cache.MakeFromRequest(proxyReq)

	return p.processHTTPRequest(w, proxyReq, key)
}

func (p *CachingMitmProxy) handleCONNECT(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("CONNECT request to %v (from %v)", proxyReq.URL, proxyReq.RemoteAddr)

	key := cache.MakeFromRequest(proxyReq)

	// "Hijack" the client connection to get a TCP (or TLS) socket we can read and write arbitrary data to/from.
	hj, ok := w.(http.Hijacker)
	if !ok {
		err := fmt.Errorf("hijacking not supported for target host '%v'. Hijacking only works with servers that support HTTP 1.x", proxyReq.URL)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	// Hijack the connection to get the underlying net.Conn.
	clientConn, _, err := hj.Hijack()
	if err != nil {
		err := fmt.Errorf("hijack failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	host, _, err := net.SplitHostPort(proxyReq.Host)
	if err != nil {
		err := fmt.Errorf("invalid host:port format %v: %v", proxyReq.Host, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// Create a fake TLS certificate for the target host, signed by our CA.
	pemCert, pemKey, err := createCert([]string{host}, p.caCert, p.caKey, 240)
	if err != nil {
		err := fmt.Errorf("failed to create TLS certificate for %v: %v", host, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		err := fmt.Errorf("failed to create X509 key pair for cert %v: %v", tlsCert, err)
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

	log.Print("Starting TLS server for CONNECT tunnel")
	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()

	// Create a buffered reader for the client connection; this is required to
	// use http package functions with this connection.
	connReader := bufio.NewReader(tlsConn)

	log.Print("Entering request loop for CONNECT tunnel")
	for {
		// Read next HTTP request from client.
		req, err := http.ReadRequest(connReader)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fmt.Errorf("error reading request from client (%v): %w", proxyReq.RemoteAddr, err)
		}

		p.processHTTPRequest(tlsConn, req, key)
	}

	return nil
}
