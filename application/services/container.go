package services

import (
	"reservoir/cache"
	"reservoir/config"
	"reservoir/db/stores"
	"reservoir/metrics"
	coreauth "reservoir/webserver/auth"
)

type CacheController interface {
	CacheStats() cache.Stats
	ClearCache() error
}

type Container struct {
	Auth    *AuthService
	Users   *UserService
	Config  *ConfigService
	Cache   *CacheService
	Metrics *metrics.Metrics
}

func NewContainer(cfg *config.Config, sessions *coreauth.SessionManager, users *stores.UserStore, cacheController CacheController) *Container {
	return NewContainerWithMetrics(cfg, sessions, users, cacheController, metrics.NewMetrics())
}

func NewContainerWithMetrics(cfg *config.Config, sessions *coreauth.SessionManager, users *stores.UserStore, cacheController CacheController, m *metrics.Metrics) *Container {
	if sessions == nil {
		sessions = coreauth.DefaultSessionManager()
	}
	m = metrics.Ensure(m)

	return &Container{
		Auth:    &AuthService{cfg: cfg, users: users, sessions: sessions, loginLimiter: coreauth.NewLoginAttemptLimiter()},
		Users:   &UserService{users: users, sessions: sessions},
		Config:  &ConfigService{cfg: cfg},
		Cache:   &CacheService{cfg: cfg, controller: cacheController},
		Metrics: m,
	}
}
