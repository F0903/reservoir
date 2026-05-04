package proxy

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reservoir/metrics"
	"reservoir/proxy/responder"
)

func (p *Proxy) handleCONNECT(r responder.Responder, proxyReq *http.Request) error {
	slog.Info("Handling CONNECT request", "url", proxyReq.URL, "remote_addr", proxyReq.RemoteAddr)

	metrics.Global.Requests.HTTPSProxyRequests.Increment()

	clientConn, _, err := r.Hijack()
	if err != nil {
		r.WriteError("Unable to take over socket.", http.StatusInternalServerError)
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

	if err := tlsConn.Handshake(); err != nil {
		slog.Error("TLS handshake failed in CONNECT tunnel", "host", proxyReq.Host, "error", err)
		return err
	}

	// Create a buffered reader for the client connection. This is required to
	// use http package functions with this connection.
	connReader := bufio.NewReader(tlsConn)
	responder := responder.NewRawHTTPResponder(tlsConn)

	slog.Debug("Entering request loop for CONNECT tunnel", "host", proxyReq.Host)
	for {
		// Read next HTTP request from client.
		req, err := http.ReadRequest(connReader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Debug("Client closed connection in CONNECT tunnel", "host", proxyReq.Host)
			} else {
				slog.Error("Error reading request from client in CONNECT tunnel", "host", proxyReq.Host, "error", err)
			}
			break
		}

		req.Close = true
		if err := p.handleHTTP(responder, req); err != nil {
			slog.Error("Error processing HTTP request in CONNECT tunnel", "host", proxyReq.Host, "error", err)
		}
	}

	slog.Debug("Exiting CONNECT tunnel", "host", proxyReq.Host)
	return nil
}
