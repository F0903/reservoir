package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"reservoir/config"
	"reservoir/logging"
	"reservoir/proxy"
	"reservoir/proxy/certs"
	"reservoir/webserver"
	"reservoir/webserver/api"
	"reservoir/webserver/dashboard"
	"syscall"
)

func startProxy(errChan chan error, ctx context.Context) error {
	cfgLock := config.Global.Immutable()

	var proxyListen, caCert, caKey, cacheDir string
	cfgLock.Read(func(c *config.Config) {
		proxyListen = c.ProxyListen.Read()
		caCert = c.CaCert.Read()
		caKey = c.CaKey.Read()
		cacheDir = c.CacheDir.Read()
	})

	ca, err := certs.NewPrivateCA(caCert, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA: %v", err)
	}

	proxy, err := proxy.New(cacheDir, ca, ctx)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %v", err)
	}

	slog.Info("Starting proxy server", "address", proxyListen)
	proxy.Listen(proxyListen, errChan, ctx)
	return nil
}

func startWebServer(errChan chan error, ctx context.Context) error {
	cfgLock := config.Global.Immutable()

	var webserverListen string
	var dashboardEnabled bool
	var apiEnabled bool
	cfgLock.Read(func(c *config.Config) {
		webserverListen = c.WebserverListen.Read()
		dashboardEnabled = c.DashboardEnabled.Read()
		apiEnabled = c.ApiEnabled.Read()
	})

	if !dashboardEnabled && !apiEnabled {
		slog.Info("Webserver is disabled by configuration, skipping startup")
		return nil
	}

	webserver := webserver.New()

	if dashboardEnabled {
		dashboard := dashboard.New()
		if err := webserver.Register(dashboard); err != nil {
			return fmt.Errorf("failed to register dashboard: %v", err)
		}
	} else {
		slog.Info("Dashboard is disabled by configuration, skipping registration")
	}

	if apiEnabled || dashboardEnabled {
		api := api.New()
		if err := webserver.Register(api); err != nil {
			return fmt.Errorf("failed to register API: %v", err)
		}
	} else {
		slog.Info("API is disabled by configuration, skipping registration")
	}

	slog.Info("Starting webserver", "address", webserverListen)
	webserver.Listen(webserverListen, errChan, ctx)
	return nil
}

func main() {
	config.ParseFlags()
	logging.Init()

	// Channel to handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := startProxy(errChan, ctx); err != nil {
		slog.Error("Failed to start proxy", "error", err)
		panic(err)
	}

	if err := startWebServer(errChan, ctx); err != nil {
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
