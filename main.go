package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"reservoir/proxy"
	"reservoir/proxy/certs"
	"reservoir/webserver"
	"reservoir/webserver/api"
	"reservoir/webserver/dashboard"
	"syscall"
)

func startProxy(address, caCertFile, caKeyFile, cacheDir string, errChan chan error, ctx context.Context) error {
	ca, err := certs.NewPrivateCA(caCertFile, caKeyFile)
	if err != nil {
		return fmt.Errorf("failed to create CA: %v", err)
	}

	proxy, err := proxy.New(cacheDir, ca, ctx)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %v", err)
	}

	slog.Info("Starting proxy server", "address", address)
	proxy.Listen(address, errChan, ctx)
	return nil
}

func startWebServer(address string, errChan chan error, ctx context.Context) error {
	webserver := webserver.New()

	dashboard := dashboard.New()
	if err := webserver.Register(dashboard); err != nil {
		return fmt.Errorf("failed to register dashboard: %v", err)
	}

	api := api.New()
	if err := webserver.Register(api); err != nil {
		return fmt.Errorf("failed to register API: %v", err)
	}

	slog.Info("Starting webserver", "address", address)
	webserver.Listen(address, errChan, ctx)
	return nil
}

func main() {
	address := flag.String("listen", ":9999", "The address and port that the proxy will listen on")
	caCertFile := flag.String("ca-cert", "ssl/ca.crt", "Path to CA certificate file")
	caKeyFile := flag.String("ca-key", "ssl/ca.key", "Path to CA private key file")
	cacheDir := flag.String("cache-dir", "var/cache/", "Path to cache directory")
	webserverAddress := flag.String("webserver-listen", "localhost:8080", "The address and port that the webserver (dashboard and API) will listen on")
	flag.Parse()

	// Channel to handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := startProxy(*address, *caCertFile, *caKeyFile, *cacheDir, errChan, ctx); err != nil {
		slog.Error("Failed to start proxy", "error", err)
		panic(err)
	}

	if err := startWebServer(*webserverAddress, errChan, ctx); err != nil {
		slog.Error("Failed to start webserver", "error", err)
		panic(err)
	}

	select {
	case err := <-errChan:
		slog.Error("Service error", "error", err)
		cancel()
		panic(err)
	case sig := <-sigChan:
		slog.Info("Received shutdown signal, shutting down gracefully...", "signal", sig)
		cancel()
	}
}
