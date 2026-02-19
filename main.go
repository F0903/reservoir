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

func startProxy(cfg *config.Config, errChan chan error, ctx context.Context) error {
	caCert := cfg.Proxy.CaCert.Read()
	caKey := cfg.Proxy.CaKey.Read()
	ca, err := certs.NewPrivateCA(caCert, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA: %v", err)
	}

	p, err := proxy.NewProxy(cfg, ca, ctx)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %v", err)
	}

	proxyListen := cfg.Proxy.Listen.Read()
	slog.Info("Starting proxy server", "address", proxyListen)
	p.Listen(proxyListen, errChan, ctx)
	return nil
}

func startWebServer(cfg *config.Config, errChan chan error, ctx context.Context) error {
	dashboardDisabled := cfg.Webserver.DashboardDisabled.Read()
	apiDisabled := cfg.Webserver.ApiDisabled.Read()
	if apiDisabled && !dashboardDisabled {
		panic("API cannot be disabled while dashboard is enabled")
	} else if apiDisabled && dashboardDisabled {
		slog.Info("Webserver is disabled by configuration, skipping startup")
		return nil
	}

	webserver := webserver.New()

	if dashboardDisabled {
		slog.Info("Dashboard is disabled by configuration, skipping registration")
	} else {
		d := dashboard.New(cfg)
		if err := webserver.Register(d); err != nil {
			return fmt.Errorf("failed to register dashboard: %v", err)
		}
	}

	if apiDisabled {
		slog.Info("API is disabled by configuration, skipping registration")
	} else {
		a := api.New(cfg)
		if err := webserver.Register(a); err != nil {
			return fmt.Errorf("failed to register API: %v", err)
		}
	}

	auth.StartSessionGC()

	webserverListen := cfg.Webserver.Listen.Read()
	slog.Info("Starting webserver", "address", webserverListen)
	webserver.Listen(webserverListen, errChan, ctx)
	return nil
}

func main() {
	config.OverrideGlobalConfigFromFlags()
	logging.Init(config.Global)

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

	if err := startProxy(config.Global, errChan, ctx); err != nil {
		slog.Error("Failed to start proxy", "error", err)
		panic(err)
	}

	if err := startWebServer(config.Global, errChan, ctx); err != nil {
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
