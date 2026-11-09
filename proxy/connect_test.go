package proxy

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reservoir/cache/memory"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/fetch"
	"strings"
	"sync"
	"testing"
	"time"
)

type selfSignedCA struct{}

func (selfSignedCA) GetCertForHost(host string) (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

func newTunnelTestProxy(t *testing.T, upstreamClient *http.Client) *Proxy {
	t.Helper()

	cfg := config.NewDefault()
	store := memory.New[fetch.CacheMetadata](cfg, metrics.NewMetrics(), 100, 1<<30, time.Hour, 8, t.Context())
	t.Cleanup(store.Destroy)

	p, err := NewProxy(cfg, selfSignedCA{}, Dependencies{Cache: store, Metrics: metrics.NewMetrics(), UpstreamClient: upstreamClient})
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	p.tunnelHandshakeTimeout = 100 * time.Millisecond
	p.tunnelIdleTimeout = 100 * time.Millisecond
	return p
}

func openTunnel(t *testing.T, proxyURL string) net.Conn {
	t.Helper()

	u, err := url.Parse(proxyURL)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	fmt.Fprintf(conn, "CONNECT example.test:443 HTTP/1.1\r\nHost: example.test:443\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("CONNECT failed: %v", err)
	}
	resp.Body.Close()
	return conn
}

func assertClosedByProxy(t *testing.T, conn net.Conn) {
	t.Helper()

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, err := conn.Read(make([]byte, 1))
	if errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatal("proxy never closed the idle tunnel")
	}
	if err == nil {
		t.Fatal("expected the connection to be closed")
	}
}

type failingCA struct{}

func (failingCA) GetCertForHost(string) (*tls.Certificate, error) {
	return nil, errors.New("no certificate")
}

func TestTunnelCertificateFailureIsCounted(t *testing.T) {
	p := newTunnelTestProxy(t, nil)
	p.ca = failingCA{}
	server := httptest.NewServer(p)
	t.Cleanup(server.Close)

	openTunnel(t, server.URL)

	if got := p.metrics.Requests.StatusServerErrorResponses.Get(); got != 1 {
		t.Fatalf("expected one server error, got %d", got)
	}
	if got := p.metrics.Requests.ClientResponses.Get(); got != 1 {
		t.Fatalf("expected one client response, got %d", got)
	}
}

func TestTunnelClosesWhenClientNeverCompletesHandshake(t *testing.T) {
	server := httptest.NewServer(newTunnelTestProxy(t, nil))
	t.Cleanup(server.Close)

	assertClosedByProxy(t, openTunnel(t, server.URL))
}

func TestTunnelClosesIdleConnection(t *testing.T) {
	p := newTunnelTestProxy(t, nil)
	p.tunnelHandshakeTimeout = time.Minute // So only the idle deadline can close the connection.

	server := httptest.NewServer(p)
	t.Cleanup(server.Close)

	conn := tls.Client(openTunnel(t, server.URL), &tls.Config{InsecureSkipVerify: true})
	if err := conn.Handshake(); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}

	assertClosedByProxy(t, conn)
}

// endlessHandler streams until the proxy hangs up, then closes the returned channel.
func endlessHandler() (http.Handler, <-chan struct{}) {
	stopped := make(chan struct{})
	var once sync.Once
	chunk := make([]byte, 64*1024)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer once.Do(func() { close(stopped) })
		w.Header().Set("Cache-Control", "no-store")
		for {
			if _, err := w.Write(chunk); err != nil {
				return
			}
			w.(http.Flusher).Flush()
		}
	}), stopped
}

func assertStalledClientDropped(t *testing.T, stopped <-chan struct{}) {
	t.Helper()

	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("proxy kept the download open for a client that stopped reading")
	}
}

func TestPlainRequestDropsClientThatStopsReading(t *testing.T) {
	handler, stopped := endlessHandler()
	upstream := httptest.NewServer(handler)
	t.Cleanup(upstream.Close)

	p := newTunnelTestProxy(t, upstream.Client())
	p.clientWriteTimeout = 200 * time.Millisecond
	server := httptest.NewServer(p)
	t.Cleanup(server.Close)

	conn, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	// Authorization skips the cache, so the proxy streams the body straight to this client.
	fmt.Fprintf(conn, "GET %s/stream HTTP/1.1\r\nHost: %s\r\nAuthorization: x\r\n\r\n", upstream.URL, strings.TrimPrefix(upstream.URL, "http://"))

	assertStalledClientDropped(t, stopped)
}

func TestTunnelDropsClientThatStopsReading(t *testing.T) {
	handler, stopped := endlessHandler()
	upstream := httptest.NewTLSServer(handler)
	t.Cleanup(upstream.Close)

	p := newTunnelTestProxy(t, upstream.Client())
	p.clientWriteTimeout = 200 * time.Millisecond
	server := httptest.NewServer(p)
	t.Cleanup(server.Close)

	conn := tls.Client(openTunnel(t, server.URL), &tls.Config{InsecureSkipVerify: true})
	if err := conn.Handshake(); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}
	fmt.Fprintf(conn, "GET /stream HTTP/1.1\r\nHost: %s\r\nAuthorization: x\r\n\r\n", strings.TrimPrefix(upstream.URL, "https://"))

	assertStalledClientDropped(t, stopped)
}

func TestDownloadSurvivesPausesLongerThanWriteTimeout(t *testing.T) {
	const chunks, chunkSize = 4, 64 * 1024

	pausing := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		for range chunks {
			w.Write(make([]byte, chunkSize))
			w.(http.Flusher).Flush()
			time.Sleep(300 * time.Millisecond)
		}
	})

	tests := []struct {
		name        string
		newUpstream func(http.Handler) *httptest.Server
	}{
		{"plain", httptest.NewServer},
		{"tunnel", httptest.NewTLSServer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := tt.newUpstream(pausing)
			t.Cleanup(upstream.Close)

			p := newTunnelTestProxy(t, upstream.Client())
			p.clientWriteTimeout = 200 * time.Millisecond
			server := httptest.NewServer(p)
			t.Cleanup(server.Close)

			proxyURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			transport := &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			t.Cleanup(transport.CloseIdleConnections)

			req, err := http.NewRequest(http.MethodGet, upstream.URL+"/pausing", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "x") // Skips the cache, so the body is streamed as it arrives.

			resp, err := (&http.Client{Transport: transport}).Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil || len(body) != chunks*chunkSize {
				t.Fatalf("got %d bytes, err=%v", len(body), err)
			}
		})
	}
}
