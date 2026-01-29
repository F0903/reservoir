package tests

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reservoir/config"
	"reservoir/logging"
	"reservoir/proxy"
	"reservoir/proxy/certs"
	"testing"
	"time"
)

type TestEnv struct {
	Upstream    *httptest.Server
	ProxyServer *httptest.Server
	Client      *http.Client
	Proxy       *proxy.Proxy
	CacheDir    string
}

func SetupTestEnv(t testing.TB) *TestEnv {
	cacheDir := t.TempDir()

	// Setup Mock Upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("ETag", "\"test-etag\"")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	}))

	// Setup Config Defaults for Tests
	config.Global.UpstreamDefaultHttps.Overwrite(false)
	config.Global.CacheDir.Overwrite(cacheDir)
	config.Global.RetryOnRange416.Overwrite(false)
	config.Global.IgnoreCacheControl.Overwrite(false)
	config.Global.ForceDefaultCacheMaxAge.Overwrite(false)

	if _, ok := t.(*testing.B); ok {
		config.Global.LogToStdout.Overwrite(false)
	}

	logging.Init()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	p, err := proxy.NewProxy(cacheDir, &FakeCA{}, ctx)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	proxyServer := httptest.NewServer(p)

	proxyUrl, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}

	proxyClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		},
	}

	t.Cleanup(func() {
		upstream.Close()
		proxyServer.Close()
		// Give some time for async operations to finish before stopping and cleaning up
		time.Sleep(100 * time.Millisecond)
		p.Destroy()
		if transport, ok := proxyClient.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	})

	return &TestEnv{
		Upstream:    upstream,
		ProxyServer: proxyServer,
		Client:      proxyClient,
		Proxy:       p,
		CacheDir:    cacheDir,
	}
}

// GenerateTestCA generates a temporary CA for testing HTTPS MITM
func GenerateTestCA(t testing.TB) (string, string) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatalf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"reservoir-test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	dir := t.TempDir()

	certFile := filepath.Join(dir, "ca.crt")
	keyFile := filepath.Join(dir, "ca.key")

	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("failed to open ca.crt for writing: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("failed to open ca.key for writing: %v", err)
	}
	privBytes, _ := x509.MarshalPKCS8PrivateKey(priv)
	pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	keyOut.Close()

	return certFile, keyFile
}

func SetupHttpsTestEnv(t testing.TB) *TestEnv {
	certFile, keyFile := GenerateTestCA(t)
	ca, err := certs.NewPrivateCA(certFile, keyFile)
	if err != nil {
		t.Fatalf("failed to create CA: %v", err)
	}

	cacheDir := t.TempDir()

	// Mock HTTPS Upstream
	// We need a server with a cert signed by our test CA, or just use httptest.NewTLSServer
	// and make the client trust it.
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("https response body"))
	}))

	config.Global.UpstreamDefaultHttps.Overwrite(true)
	config.Global.CacheDir.Overwrite(cacheDir)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	p, err := proxy.NewProxy(cacheDir, ca, ctx)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	proxyServer := httptest.NewServer(p)

	proxyUrl, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}

	// Client must trust the proxy's CA for MITM
	caCertPool := x509.NewCertPool()
	caCertBytes, _ := os.ReadFile(certFile)
	caCertPool.AppendCertsFromPEM(caCertBytes)

	// Proxy must trust the upstream server
	if upstream.TLS != nil {
		caCertPool.AddCert(upstream.Certificate())
	}

	// Update default client to trust our CA (for the proxy -> upstream connection)
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport.TLSClientConfig = &tls.Config{
			RootCAs: caCertPool,
		}
	}

	proxyClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	t.Cleanup(func() {
		upstream.Close()
		proxyServer.Close()
		// Give some time for async operations to finish before stopping and cleaning up
		time.Sleep(100 * time.Millisecond)
		p.Destroy()
		if transport, ok := proxyClient.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	})

	return &TestEnv{
		Upstream:    upstream,
		ProxyServer: proxyServer,
		Client:      proxyClient,
		Proxy:       p,
		CacheDir:    cacheDir,
	}
}
