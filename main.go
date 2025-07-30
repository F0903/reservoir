package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"reservoir/config"
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
	config := config.Get()
	if !config.DashboardEnabled.Read() && !config.ApiEnabled.Read() {
		slog.Info("Webserver is disabled by configuration, skipping startup")
		return nil
	}

	webserver := webserver.New()

	if config.DashboardEnabled.Read() {
		dashboard := dashboard.New()
		if err := webserver.Register(dashboard); err != nil {
			return fmt.Errorf("failed to register dashboard: %v", err)
		}
	} else {
		slog.Info("Dashboard is disabled by configuration, skipping registration")
	}

	if config.ApiEnabled.Read() || config.DashboardEnabled.Read() {
		api := api.New()
		if err := webserver.Register(api); err != nil {
			return fmt.Errorf("failed to register API: %v", err)
		}
	} else {
		slog.Info("API is disabled by configuration, skipping registration")
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
	noDashboard := flag.Bool("no-dashboard", false, "Disable the dashboard")
	noApi := flag.Bool("no-api", false, "Disable the API")
	flag.Parse()

	if *noDashboard || *noApi {
		slog.Info("Updating global config based on command line flags", "dashboard_disabled", *noDashboard, "api_disabled", *noApi)
		config.Update(func(cfg *config.Config) {
			if *noDashboard {
				cfg.DashboardEnabled.Overwrite(false)
			}

			if *noApi {
				cfg.ApiEnabled.Overwrite(false)
			}
		})
	}

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
