package tests

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestHttpsMITM(t *testing.T) {
	env := SetupHttpsTestEnv(t)
	env.Start()

	targetURL := env.Upstream.URL + "/https-test"
	resp, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("Failed to make HTTPS request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "https response body" {
		t.Errorf("Expected response body 'https response body', got '%s'", string(body))
	}
}

func TestConnectTunnelRepeatedRequestsDoNotLeakResponseHeaders(t *testing.T) {
	env := SetupHttpsTestEnv(t)
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")

		switch r.URL.Path {
		case "/first":
			w.Header().Set("X-First-Only", "first")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("first body"))
		case "/second":
			w.Header().Set("X-Second-Only", "second")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("second body"))
		case "/third":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("third body"))
		default:
			http.NotFound(w, r)
		}
	})
	env.Start()

	proxyURL := mustParseURL(t, env.ProxyServer.URL)
	targetURL := mustParseURL(t, env.Upstream.URL)
	targetHost := targetURL.Host

	conn, err := net.Dial("tcp", proxyURL.Host)
	if err != nil {
		t.Fatalf("failed to dial proxy: %v", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("failed to set tunnel deadline: %v", err)
	}

	if _, err := fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", targetHost, targetHost); err != nil {
		t.Fatalf("failed to write CONNECT request: %v", err)
	}

	connectResp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("failed to read CONNECT response: %v", err)
	}
	if connectResp.StatusCode != http.StatusOK {
		t.Fatalf("expected CONNECT 200 OK, got %d", connectResp.StatusCode)
	}
	connectResp.Body.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		RootCAs:    env.CACertPool,
		ServerName: targetURL.Hostname(),
		MinVersion: tls.VersionTLS12,
	})
	defer tlsConn.Close()
	if err := tlsConn.Handshake(); err != nil {
		t.Fatalf("failed to complete tunnel TLS handshake: %v", err)
	}

	reader := bufio.NewReader(tlsConn)
	first := roundTripTunnelRequest(t, tlsConn, reader, targetHost, "/first")
	second := roundTripTunnelRequest(t, tlsConn, reader, targetHost, "/second")
	third := roundTripTunnelRequest(t, tlsConn, reader, targetHost, "/third")

	assertTunnelResponse(t, first, "first body", "X-First-Only", "first", "X-Second-Only")
	assertTunnelResponse(t, second, "second body", "X-Second-Only", "second", "X-First-Only")
	assertTunnelResponse(t, third, "third body", "", "", "X-First-Only", "X-Second-Only")
}

type tunnelResponse struct {
	StatusCode int
	Header     http.Header
	Body       string
}

func roundTripTunnelRequest(t *testing.T, conn net.Conn, reader *bufio.Reader, targetHost string, path string) tunnelResponse {
	t.Helper()

	if _, err := fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nAccept-Encoding: identity\r\nConnection: keep-alive\r\n\r\n", path, targetHost); err != nil {
		t.Fatalf("failed to write tunneled request %s: %v", path, err)
	}

	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("failed to read tunneled response %s: %v", path, err)
	}

	return tunnelResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
		Body:       readResponseBody(t, resp),
	}
}

func assertTunnelResponse(t *testing.T, resp tunnelResponse, wantBody string, wantHeader string, wantHeaderValue string, absentHeaders ...string) {
	t.Helper()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected tunneled response 200 OK, got %d", resp.StatusCode)
	}
	if resp.Body != wantBody {
		t.Fatalf("expected tunneled body %q, got %q", wantBody, resp.Body)
	}
	if wantHeader != "" {
		if got := resp.Header.Get(wantHeader); got != wantHeaderValue {
			t.Fatalf("expected %s=%q, got %q", wantHeader, wantHeaderValue, got)
		}
	}
	for _, header := range absentHeaders {
		if got := resp.Header.Get(header); got != "" {
			t.Fatalf("expected %s to be absent, got %q", header, got)
		}
	}
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", rawURL, err)
	}
	return parsed
}
