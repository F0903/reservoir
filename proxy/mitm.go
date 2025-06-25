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
	"net/url"
	"strings"
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
		err_msg := fmt.Sprintf("Failed to load CA certificate and key: %v", err)
		return nil, errors.New(err_msg)
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
	// remove proxy headers
	proxyReq.RequestURI = ""
	transport := http.DefaultTransport
	resp, err := transport.RoundTrip(proxyReq)
	if err != nil {
		err_msg := fmt.Sprintf("error forwarding request to target %v: %v", proxyReq.Host, err)
		http.Error(w, err_msg, http.StatusBadGateway)
		return errors.New(err_msg)
	}
	defer resp.Body.Close()

	// copy headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	return nil
}

func (p *MitmProxy) handleCONNECT(w http.ResponseWriter, proxyReq *http.Request) error {
	log.Printf("CONNECT request to %v (from %v)", proxyReq.Host, proxyReq.RemoteAddr)

	// "Hijack" the client connection to get a TCP (or TLS) socket we can read and write arbitrary data to/from.
	hj, ok := w.(http.Hijacker)
	if !ok {
		err_msg := fmt.Sprintf("hijacking not supported for target host '%v'. Hijacking only works with servers that support HTTP 1.x", proxyReq.Host)
		http.Error(w, err_msg, http.StatusInternalServerError)
		return errors.New(err_msg)
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		err_msg := fmt.Sprintf("hijack failed: %v", err)
		http.Error(w, err_msg, http.StatusInternalServerError)
		return errors.New(err_msg)
	}

	host, _, err := net.SplitHostPort(proxyReq.Host)
	if err != nil {
		err_msg := fmt.Sprintf("invalid host:port format %v: %v", proxyReq.Host, err)
		http.Error(w, err_msg, http.StatusBadRequest)
		return errors.New(err_msg)
	}

	// Create a fake TLS certificate for the target host, signed by our CA.
	pemCert, pemKey := createCert([]string{host}, p.caCert, p.caKey, 240)
	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		err_msg := fmt.Sprintf("failed to create TLS certificate for %v: %v", host, err)
		return errors.New(err_msg)
	}

	// Send an HTTP OK response back to the client; this initiates the CONNECT
	// tunnel. From this point on the client will assume it's connected directly
	// to the target.
	if _, err := clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		err_msg := fmt.Sprintf("failed to write HTTP OK response to client: %v", err)
		return errors.New(err_msg)
	}

	// Configure a new TLS server, pointing it at the client connection, using
	// our certificate. This server will now pretend being the target.
	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:       tls.VersionTLS12,
		Certificates:     []tls.Certificate{tlsCert},
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()
	proxyRequestLoop(tlsConn, proxyReq)

	return nil
}

func proxyRequestLoop(tlsConn *tls.Conn, proxyReq *http.Request) {
	// Create a buffered reader for the client connection; this is required to
	// use http package functions with this connection.
	connReader := bufio.NewReader(tlsConn)

	for {
		// Read next HTTP request from client.
		req, err := http.ReadRequest(connReader)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		// We can dump the request; log it, modify it...
		if b, err := httputil.DumpRequest(req, false); err == nil {
			log.Printf("incoming request:\n%s\n", string(b))
		}

		// Take the request and changes its destination to be forwarded to the target server.
		// The target server is specified in the original CONNECT request, which is proxyReq.
		changeRequestToTarget(req, proxyReq.Host)

		// Send the request to the target server and log the response.
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalf("error sending request to '%v': %v", proxyReq.Host, err)
		}
		defer resp.Body.Close()
		log.Printf("sent request to target %v, got response status: %s", proxyReq.Host, resp.Status)

		// Send the target server's response back to the client.
		if err := resp.Write(tlsConn); err != nil {
			log.Printf("error writing response back to client (%v): %v\n", proxyReq.RemoteAddr, err)
		}
	}
}

func changeRequestToTarget(req *http.Request, targetHost string) {
	targetUrl, err := addrToUrl(targetHost)
	if err != nil {
		log.Fatalf("invalid target host '%s': %v", targetHost, err)
	}

	targetUrl.Path = req.URL.Path
	targetUrl.RawQuery = req.URL.RawQuery
	req.URL = targetUrl
	// Make sure this is unset for sending the request through a client
	req.RequestURI = ""
}

func addrToUrl(addr string) (*url.URL, error) {
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "https://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	return u, nil
}
