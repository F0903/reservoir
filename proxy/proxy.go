package proxy

import (
	"context"
	"errors"
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/certs"
	"reservoir/proxy/fetch"
	"reservoir/utils/httplistener"
	"time"
)

var (
	ErrTLSCertFailed        = errors.New("error getting TLS certificate")
	ErrClientResponseFailed = errors.New("failed to write HTTP OK response to client")
	ErrRangeNotSatisfiable  = errors.New("range not satisfiable")
	ErrIfRangeMismatch      = errors.New("If-Range header mismatch")
	ErrCacheStoreRequired   = errors.New("proxy cache store is required")
)

type Proxy struct {
	ca      certs.CertAuthority
	cache   cache.Cache[fetch.CacheMetadata]
	fetcher *fetch.Fetcher
	cfg     *config.Config
	metrics *metrics.Metrics

	tunnelHandshakeTimeout time.Duration
	tunnelIdleTimeout      time.Duration
	clientWriteTimeout     time.Duration
}

type Dependencies struct {
	Cache          cache.Cache[fetch.CacheMetadata]
	Metrics        *metrics.Metrics
	UpstreamClient *http.Client
}

func (p *Proxy) Run(address string, ctx context.Context) error {
	listener := httplistener.New(address, p)
	return listener.Run(ctx)
}

func (p *Proxy) Destroy() {
	p.fetcher.CloseIdleConnections()
	p.cache.Destroy()
}

func (p *Proxy) CacheStats() cache.Stats {
	return p.cache.Stats()
}

func (p *Proxy) ClearCache() error {
	return p.cache.Clear()
}

func NewProxy(cfg *config.Config, ca certs.CertAuthority, deps Dependencies) (*Proxy, error) {
	if deps.Cache == nil {
		return nil, ErrCacheStoreRequired
	}

	m := metrics.Ensure(deps.Metrics)
	return &Proxy{
		ca:      ca,
		cache:   deps.Cache,
		fetcher: fetch.New(deps.Cache, cfg, deps.UpstreamClient, m),
		cfg:     cfg,
		metrics: m,

		tunnelHandshakeTimeout: defaultTunnelHandshakeTimeout,
		tunnelIdleTimeout:      defaultTunnelIdleTimeout,
		clientWriteTimeout:     defaultClientWriteTimeout,
	}, nil
}
