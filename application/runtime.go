package application

import (
	"context"
	"fmt"
	"log/slog"
	appservices "reservoir/application/services"
	"reservoir/config"
	"reservoir/db"
	"reservoir/db/stores"
	"reservoir/metrics"
	"reservoir/proxy"
	"reservoir/proxy/certs"
	"reservoir/webserver"
	"reservoir/webserver/api"
	"reservoir/webserver/auth"
	"reservoir/webserver/dashboard"

	"golang.org/x/sync/errgroup"
)

const databaseBusyTimeout = 5000

type Runtime struct {
	cfg              *config.Config
	proxy            *proxy.Proxy
	webserver        *webserver.WebServer
	sessions         *auth.SessionManager
	userStore        *stores.UserStore
	services         *appservices.Container
	metrics          *metrics.Metrics
	webserverEnabled bool
	sessionGCEnabled bool
}

func NewRuntime(cfg *config.Config, ctx context.Context) (*Runtime, error) {
	apiDisabled := cfg.Webserver.ApiDisabled.Read()
	runtimeMetrics := metrics.NewMetrics()

	var users *stores.UserStore
	if !apiDisabled {
		databasePath := cfg.Database.File.Read()
		database, err := db.Open(databasePath, databaseBusyTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to open main database %q: %w", databasePath, err)
		}
		if err := database.Migrate(); err != nil {
			_ = database.Close()
			return nil, fmt.Errorf("failed to migrate main database %q: %w", databasePath, err)
		}
		users = stores.NewUserStore(database)

		bootstrap, err := auth.EnsureBootstrapAdmin(users)
		if err != nil {
			_ = users.Close()
			return nil, fmt.Errorf("failed to ensure bootstrap admin: %w", err)
		}
		if bootstrap != nil {
			if bootstrap.Required {
				slog.Warn("Initial admin setup required", "bootstrap_path", "/bootstrap")
			}
		}
	}

	caCert := cfg.Proxy.CaCert.Read()
	caKey := cfg.Proxy.CaKey.Read()
	ca, err := certs.NewPrivateCA(caCert, caKey)
	if err != nil {
		if users != nil {
			_ = users.Close()
		}
		return nil, fmt.Errorf("failed to create CA: %w", err)
	}

	cacheStore, err := NewProxyCache(cfg, runtimeMetrics, ctx)
	if err != nil {
		if users != nil {
			_ = users.Close()
		}
		return nil, fmt.Errorf("failed to create proxy cache: %w", err)
	}

	p, err := proxy.NewProxy(cfg, ca, proxy.Dependencies{
		Cache:   cacheStore,
		Metrics: runtimeMetrics,
	})
	if err != nil {
		cacheStore.Destroy()
		if users != nil {
			_ = users.Close()
		}
		return nil, fmt.Errorf("failed to create proxy: %w", err)
	}

	sessions := auth.NewSessionManager()
	svc := appservices.NewContainerWithMetrics(cfg, sessions, users, p, runtimeMetrics)
	ws, webserverEnabled, sessionGCEnabled, err := buildWebServer(cfg, svc)
	if err != nil {
		p.Destroy()
		if users != nil {
			_ = users.Close()
		}
		return nil, err
	}

	return &Runtime{
		cfg:              cfg,
		proxy:            p,
		webserver:        ws,
		sessions:         sessions,
		userStore:        users,
		services:         svc,
		metrics:          runtimeMetrics,
		webserverEnabled: webserverEnabled,
		sessionGCEnabled: sessionGCEnabled,
	}, nil
}

func buildWebServer(cfg *config.Config, services *appservices.Container) (*webserver.WebServer, bool, bool, error) {
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
		if services == nil || services.Auth == nil || services.Users == nil {
			return nil, false, false, fmt.Errorf("API requires initialized application services")
		}
		a := api.New(services)
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
			sessions := r.sessions
			if sessions == nil {
				sessions = auth.DefaultSessionManager()
			}
			return sessions.RunGC(groupCtx)
		})
	}

	return group.Wait()
}

func (r *Runtime) Close() {
	if r.proxy != nil {
		r.proxy.Destroy()
	}
	if r.userStore != nil {
		if err := r.userStore.Close(); err != nil {
			slog.Error("Failed to close user store", "error", err)
		}
	}
}
