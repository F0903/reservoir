package proxy

import (
	"context"
	"errors"
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/certs"
	"reservoir/utils/httplistener"
	"time"
)

var (
	ErrCacheGetFailed       = errors.New("error getting cache for key")
	ErrNoCachedResponse304  = errors.New("received 304 Not Modified but no cached response found")
	ErrUpdateCacheMetadata  = errors.New("error updating cache metadata")
	ErrCacheResponseFailed  = errors.New("error caching response")
	ErrTLSCertFailed        = errors.New("error getting TLS certificate")
	ErrClientResponseFailed = errors.New("failed to write HTTP OK response to client")
	ErrReadRequestFailed    = errors.New("error reading request from client")
	ErrRangeNotSatisfiable  = errors.New("range not satisfiable")
	ErrIfRangeMismatch      = errors.New("If-Range header mismatch")
	ErrBadGateway           = errors.New("bad gateway. Error when sending request to upstream")
	ErrCacheStoreRequired   = errors.New("proxy cache store is required")
)

type CacheMetadata struct {
	ETag         string
	LastModified time.Time
	Header       http.Header
	Vary         []string
}

type Proxy struct {
	ca      certs.CertAuthority
	cache   cache.Cache[CacheMetadata]
	fetch   fetcher
	cfg     *config.Config
	metrics *metrics.Metrics
}

type Dependencies struct {
	Cache          cache.Cache[CacheMetadata]
	Metrics        *metrics.Metrics
	UpstreamClient *http.Client
}

func (p *Proxy) Listen(address string, errChan chan error, ctx context.Context) {
	listener := httplistener.New(address, p)
	listener.ListenWithCancel(errChan, ctx)
}

func (p *Proxy) Run(address string, ctx context.Context) error {
	listener := httplistener.New(address, p)
	return listener.Run(ctx)
}

func (p *Proxy) Destroy() {
	p.fetch.closeIdleConnections()
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
		fetch:   newFetcher(deps.Cache, cfg, deps.UpstreamClient, m),
		cfg:     cfg,
		metrics: m,
	}, nil
}
