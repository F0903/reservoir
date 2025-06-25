package proxy

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
)

type MitmProxy struct {
	caCert *x509.Certificate
	caKey  any
}

// createMitmProxy creates a new MITM proxy. It should be passed the filenames
// for the certificate and private key of a certificate authority trusted by the
// client's machine.
func NewMitmProxy(caCertFile, caKeyFile string) (*MitmProxy, error) {
	caCert, caKey, err := loadX509KeyPair(caCertFile, caKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate and key: %v", err)
	}
	log.Printf("Loaded CA certificate: %v (IsCA=%v)\n", caCert, caCert.IsCA)

	return &MitmProxy{
		caCert: caCert,
		caKey:  caKey,
	}, nil
}

func (p *MitmProxy) ServeHTTP(w http.ResponseWriter, proxyReq *http.Request) {
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

func (p *MitmProxy) handleHTTP(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("HTTP request to %v (from %v)", proxyReq.Host, proxyReq.RemoteAddr)

	// remove proxy headers
	proxyReq.RequestURI = ""
	transport := http.DefaultTransport

	// Remove hop-by-hop headers that should not be forwarded to the target server.
	removeHopByHopHeaders(proxyReq.Header)
	resp, err := transport.RoundTrip(proxyReq)
	if err != nil {
		err := fmt.Errorf("error forwarding request to target %v: %v", proxyReq.Host, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close()

	// Copy headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	// Remove any hop-by-hop headers that should not be forwarded to the client.
	removeHopByHopHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	return nil
}

func (p *MitmProxy) handleCONNECT(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("CONNECT request to %v (from %v)", proxyReq.Host, proxyReq.RemoteAddr)

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
	proxyRequestLoop(tlsConn, proxyReq)
	tlsConn.Close()

	return nil
}

func proxyRequestLoop(tlsConn *tls.Conn, proxyReq *http.Request) error {
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

		// We can dump the request; log it, modify it...
		if b, err := httputil.DumpRequest(req, false); err == nil {
			log.Printf("incoming request:\n%s\n", string(b))
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

		// Send the target server's response back to the client.
		if err := resp.Write(tlsConn); err != nil {
			log.Printf("error writing response back to client (%v): %v\n", proxyReq.RemoteAddr, err)
		}
		resp.Body.Close() // Don't defer this, as it will only close the body when the function returns, not after each iteration.
	}

	return nil
}
