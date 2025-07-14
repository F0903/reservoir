package main

import (
	"apt_cacher_go/proxy"
	"apt_cacher_go/proxy/certs"
	"apt_cacher_go/webserver"
	"apt_cacher_go/webserver/api"
	"apt_cacher_go/webserver/dashboard"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func startProxy(address, caCertFile, caKeyFile, cacheDir string, errChan chan error, ctx context.Context) error {
	ca, err := certs.NewPrivateCA(caCertFile, caKeyFile)
	if err != nil {
		return fmt.Errorf("Failed to create CA: %v", err)
	}

	proxy, err := proxy.New(cacheDir, ca)
	if err != nil {
		return fmt.Errorf("Failed to create proxy: %v", err)
	}

	log.Println("Starting proxy server on", address)
	proxy.Listen(address, errChan, ctx)
	return nil
}

func startWebServer(address string, errChan chan error, ctx context.Context) error {
	webserver := webserver.New()

	dashboard := dashboard.New()
	if err := webserver.Register(dashboard); err != nil {
		return fmt.Errorf("Failed to register dashboard: %v", err)
	}

	api := api.New()
	if err := webserver.Register(api); err != nil {
		return fmt.Errorf("Failed to register API: %v", err)
	}

	log.Println("Starting webserver server on", address)
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
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := startProxy(*address, *caCertFile, *caKeyFile, *cacheDir, errChan, ctx); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}

	if err := startWebServer(*webserverAddress, errChan, ctx); err != nil {
		log.Fatalf("Failed to start webserver: %v", err)
	}

	select {
	case err := <-errChan:
		log.Fatalf("Service error: %v", err)
		cancel()
	case sig := <-sigChan:
		log.Printf("Received signal %s, shutting down...", sig)
		cancel()
	}
}
