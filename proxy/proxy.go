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
	"time"
)

type cachedRequestInfo struct {
	ETag string
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
	log.Printf("Loaded CA certificate: %v (IsCA=%v)\n", caCert, caCert.IsCA)

	return &CachingMitmProxy{
		caCert:         caCert,
		caKey:          caKey,
		cache:          cache.NewFileCache[cachedRequestInfo](cacheDir),
		defaultExpires: 1 * time.Hour,
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

func (p *CachingMitmProxy) parseExpiresOrDefault(header http.Header) time.Time {
	expires := header.Get("Expires")
	expiresTime, err := time.Parse(time.RFC1123, expires)
	if err != nil {
		log.Printf("Failed to parse Expires header: %v, using default expiration time", err)
		expiresTime = time.Now().Add(p.defaultExpires)
	}
	return expiresTime
}

// Checks if the request is already cached. If it is, it returns the cached entry.
// If the entry is expired, it removes it from the cache and returns nil.
func (p *CachingMitmProxy) getCached(key string) (*cache.Entry[cachedRequestInfo], error) {
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

func (p *CachingMitmProxy) handleHTTP(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("HTTP request to %v (from %v)", proxyReq.Host, proxyReq.RemoteAddr)

	key := buildCacheKeyFromRequest(proxyReq)

	if cached, err := p.getCached(key); cached != nil && err == nil {
		log.Printf("Serving cached response for %v", key)
		io.Copy(w, cached.Data)
		return err
	} else if err != nil {
		err := fmt.Errorf("error getting cache for %v: %v", key, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	// remove proxy headers
	proxyReq.RequestURI = ""
	transport := http.DefaultTransport

	// Remove hop-by-hop headers that should not be forwarded to the target server.
	removeHopByHopHeaders(proxyReq.Header)
	log.Printf("Forwarding request to target %v", proxyReq.Host)
	resp, err := transport.RoundTrip(proxyReq)
	if err != nil {
		err := fmt.Errorf("error forwarding request to target %v: %v", proxyReq.Host, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close()

	log.Printf("Received response from target %v, status: %s", proxyReq.Host, resp.Status)

	// Copy headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	// Remove any hop-by-hop headers that should not be forwarded to the client.
	removeHopByHopHeaders(w.Header())

	expiresTime := p.parseExpiresOrDefault(resp.Header)
	etag := resp.Header.Get("ETag")

	p.cache.Cache(key, resp.Body, expiresTime, cachedRequestInfo{
		ETag: etag,
	}) // Cache the response for 1 hour

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	return nil
}

func (p *CachingMitmProxy) handleCONNECT(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("CONNECT request to %v (from %v)", proxyReq.Host, proxyReq.RemoteAddr)

	key := buildCacheKeyFromRequest(proxyReq)

	// "Hijack" the client connection to get a TCP (or TLS) socket we can read and write arbitrary data to/from.
	hj, ok := w.(http.Hijacker)
	if !ok {
		err := fmt.Errorf("hijacking not supported for target host '%v'. Hijacking only works with servers that support HTTP 1.x", proxyReq.Host)
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
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:       tls.VersionTLS12,
		Certificates:     []tls.Certificate{tlsCert},
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

		log.Printf("Received request from client (%v): %s %s", proxyReq.RemoteAddr, req.Method, req.URL)

		if cached, err := p.getCached(key); cached != nil && err == nil {
			log.Printf("Serving cached response for %v", req.Host)
			io.Copy(tlsConn, cached.Data)
			cached.Data.Close() // Close the cached response body
			return err
		} else if err != nil {
			err := fmt.Errorf("error getting cache for %v: %v", req.Host, err)
			writeRawHTTPResonse(tlsConn, http.StatusInternalServerError, err.Error())
			return err
		}

		// Take the request and changes its destination to be forwarded to the target server.
		// The target server is specified in the original CONNECT request, which is proxyReq.
		changeRequestToTarget(req, proxyReq.Host)

		// Remove hop-by-hop headers in the request that should not be forwarded to the target server.
		removeHopByHopHeaders(req.Header)

		// Send the request to the target server and log the response.
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("error sending request to target (%v): %w", proxyReq.Host, err)
		}
		log.Printf("sent request to target %v, got response status: %s", proxyReq.Host, resp.Status)

		// Remove any hop-by-hop headers in the response that should not be forwarded to the client.
		removeHopByHopHeaders(resp.Header)

		expiresTime := p.parseExpiresOrDefault(resp.Header)
		etag := resp.Header.Get("ETag")
		p.cache.Cache(key, resp.Body, expiresTime, cachedRequestInfo{
			ETag: etag,
		})

		// Send the target server's response back to the client.
		if err := resp.Write(tlsConn); err != nil {
			log.Printf("error writing response back to client (%v): %v\n", proxyReq.RemoteAddr, err)
		}
		resp.Body.Close() // Don't defer this, as it will only close the body when the function returns, not after each iteration.
	}

	return nil
}
