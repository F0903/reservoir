package services

import (
	"reservoir/cache"
	"reservoir/config"
	"reservoir/db/stores"
	coreauth "reservoir/webserver/auth"
)

type CacheController interface {
	CacheStats() cache.Stats
	ClearCache() error
}

type Container struct {
	Auth   *AuthService
	Users  *UserService
	Config *ConfigService
	Cache  *CacheService
}

func NewContainer(cfg *config.Config, sessions *coreauth.SessionManager, users *stores.UserStore, cacheController CacheController) *Container {
	if sessions == nil {
		sessions = coreauth.DefaultSessionManager()
	}

	return &Container{
		Auth:   &AuthService{users: users, sessions: sessions},
		Users:  &UserService{users: users, sessions: sessions},
		Config: &ConfigService{cfg: cfg},
		Cache:  &CacheService{cfg: cfg, controller: cacheController},
	}
}
