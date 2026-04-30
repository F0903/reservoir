package main

import (
	"context"
	"crypto/tls"
	"net"
	"reservoir/config"
	"reservoir/proxy"
	"reservoir/webserver"
	"testing"
	"time"
)

type runtimeTestCA struct{}

func (runtimeTestCA) GetCertForHost(string) (*tls.Certificate, error) {
	return nil, nil
}

func TestRuntimeRunStopsServicesOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.NewDefault()
	cfg.Proxy.Listen.Overwrite(freeTCPAddress(t))
	cfg.Proxy.UpstreamDefaultHttps.Overwrite(false)
	cfg.Webserver.Listen.Overwrite(freeTCPAddress(t))
	cfg.Cache.Type.Overwrite(config.CacheTypeMemory)
	cfg.Cache.File.Dir.Overwrite(t.TempDir())
	cfg.Cache.LockShards.Overwrite(32)

	p, err := proxy.NewProxy(cfg, runtimeTestCA{}, ctx)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	rt := &Runtime{
		cfg:              cfg,
		proxy:            p,
		webserver:        webserver.New(),
		webserverEnabled: true,
		sessionGCEnabled: true,
	}
	defer rt.Close()

	done := make(chan error, 1)
	go func() {
		done <- rt.Run(ctx)
	}()

	waitForRuntimeTCP(t, cfg.Proxy.Listen.Read(), done)
	waitForRuntimeTCP(t, cfg.Webserver.Listen.Read(), done)

	select {
	case err := <-done:
		t.Fatalf("Runtime.Run returned before cancellation: %v", err)
	default:
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Runtime.Run returned error after cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Runtime.Run did not stop after context cancellation")
	}
}

func freeTCPAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve TCP address: %v", err)
	}
	defer listener.Close()

	return listener.Addr().String()
}

func waitForRuntimeTCP(t *testing.T, address string, done <-chan error) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("Runtime.Run returned before listener %s was ready: %v", address, err)
		default:
		}

		conn, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("listener %s did not start before timeout", address)
}
