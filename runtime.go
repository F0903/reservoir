package main

import (
	"context"
	"fmt"
	"log/slog"
	"reservoir/config"
	"reservoir/db"
	"reservoir/proxy"
	"reservoir/proxy/certs"
	"reservoir/webserver"
	"reservoir/webserver/api"
	"reservoir/webserver/auth"
	"reservoir/webserver/dashboard"

	"golang.org/x/sync/errgroup"
)

type Runtime struct {
	cfg              *config.Config
	proxy            *proxy.Proxy
	webserver        *webserver.WebServer
	webserverEnabled bool
	sessionGCEnabled bool
}

func NewRuntime(cfg *config.Config, ctx context.Context) (*Runtime, error) {
	if err := db.MigrateDatabases(); err != nil {
		return nil, fmt.Errorf("failed to migrate databases: %w", err)
	}

	caCert := cfg.Proxy.CaCert.Read()
	caKey := cfg.Proxy.CaKey.Read()
	ca, err := certs.NewPrivateCA(caCert, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA: %w", err)
	}

	p, err := proxy.NewProxy(cfg, ca, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy: %w", err)
	}

	ws, webserverEnabled, sessionGCEnabled, err := buildWebServer(cfg)
	if err != nil {
		p.Destroy()
		return nil, err
	}

	return &Runtime{
		cfg:              cfg,
		proxy:            p,
		webserver:        ws,
		webserverEnabled: webserverEnabled,
		sessionGCEnabled: sessionGCEnabled,
	}, nil
}

func buildWebServer(cfg *config.Config) (*webserver.WebServer, bool, bool, error) {
	dashboardDisabled := cfg.Webserver.DashboardDisabled.Read()
	apiDisabled := cfg.Webserver.ApiDisabled.Read()

	if apiDisabled && !dashboardDisabled {
		return nil, false, false, fmt.Errorf("API cannot be disabled while dashboard is enabled")
	}
	if apiDisabled && dashboardDisabled {
		slog.Info("Webserver is disabled by configuration, skipping startup")
		return nil, false, false, nil
	}

	ws := webserver.New()

	if dashboardDisabled {
		slog.Info("Dashboard is disabled by configuration, skipping registration")
	} else {
		d := dashboard.New(cfg)
		if err := ws.Register(d); err != nil {
			return nil, false, false, fmt.Errorf("failed to register dashboard: %w", err)
		}
	}

	if apiDisabled {
		slog.Info("API is disabled by configuration, skipping registration")
	} else {
		a := api.New(cfg)
		if err := ws.Register(a); err != nil {
			return nil, false, false, fmt.Errorf("failed to register API: %w", err)
		}
	}

	return ws, true, !apiDisabled, nil
}

func (r *Runtime) Run(ctx context.Context) error {
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		proxyListen := r.cfg.Proxy.Listen.Read()
		slog.Info("Starting proxy server", "address", proxyListen)
		return r.proxy.Run(proxyListen, groupCtx)
	})

	if r.webserverEnabled {
		group.Go(func() error {
			webserverListen := r.cfg.Webserver.Listen.Read()
			slog.Info("Starting webserver", "address", webserverListen)
			return r.webserver.Run(webserverListen, groupCtx)
		})
	}

	if r.sessionGCEnabled {
		group.Go(func() error {
			return auth.RunSessionGC(groupCtx)
		})
	}

	return group.Wait()
}

func (r *Runtime) Close() {
	if r.proxy != nil {
		r.proxy.Destroy()
	}
}
