package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"reservoir/config"
	"reservoir/db"
	"reservoir/logging"
	"reservoir/proxy"
	"reservoir/proxy/certs"
	"reservoir/webserver"
	"reservoir/webserver/api"
	"reservoir/webserver/auth"
	"reservoir/webserver/dashboard"
	"syscall"
)

func startProxy(errChan chan error, ctx context.Context) error {
	proxyListen := config.Global.ProxyListen.Read()
	caCert := config.Global.CaCert.Read()
	caKey := config.Global.CaKey.Read()
	cacheDir := config.Global.CacheDir.Read()

	ca, err := certs.NewPrivateCA(caCert, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA: %v", err)
	}

	proxy, err := proxy.NewProxy(cacheDir, ca, ctx)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %v", err)
	}

	slog.Info("Starting proxy server", "address", proxyListen)
	proxy.Listen(proxyListen, errChan, ctx)
	return nil
}

func startWebServer(errChan chan error, ctx context.Context) error {
	webserverListen := config.Global.WebserverListen.Read()
	dashboardDisabled := config.Global.DashboardDisabled.Read()
	apiDisabled := config.Global.ApiDisabled.Read()

	if apiDisabled && !dashboardDisabled {
		panic("API cannot be disabled while dashboard is enabled")
	}

	if dashboardDisabled && apiDisabled {
		slog.Info("Webserver is disabled by configuration, skipping startup")
		return nil
	}

	webserver := webserver.New()

	if dashboardDisabled {
		slog.Info("Dashboard is disabled by configuration, skipping registration")
	} else {
		dashboard := dashboard.New()
		if err := webserver.Register(dashboard); err != nil {
			return fmt.Errorf("failed to register dashboard: %v", err)
		}
	}

	if apiDisabled {
		slog.Info("API is disabled by configuration, skipping registration")
	} else {
		api := api.New()
		if err := webserver.Register(api); err != nil {
			return fmt.Errorf("failed to register API: %v", err)
		}
	}

	auth.StartSessionGC()

	slog.Info("Starting webserver", "address", webserverListen)
	webserver.Listen(webserverListen, errChan, ctx)
	return nil
}

func main() {
	config.OverrideGlobalConfigFromFlags()
	logging.Init()

	// Channel to handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := db.MigrateDatabases(); err != nil {
		slog.Error("Failed to migrate databases", "error", err)
		panic(err)
	}

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
