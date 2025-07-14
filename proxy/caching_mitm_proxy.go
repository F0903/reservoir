package proxy

import (
	"apt_cacher_go/proxy/certs"
	"apt_cacher_go/utils/http_listener"
	"context"
)

type CachingMitmProxy struct {
	handler *cachingMitmProxyHandler
}

// Creates a new MITM proxy. It should be passed the filenames
// for the certificate and private key of a certificate authority trusted by the
// client's machine.
func New(cacheDir string, ca certs.CertAuthority) (*CachingMitmProxy, error) {
	handler, err := newCachingMitmProxyHandler(cacheDir, ca)
	if err != nil {
		return nil, err
	}
	return &CachingMitmProxy{
		handler: handler,
	}, nil
}

func (p *CachingMitmProxy) Listen(address string, errChan chan error, ctx context.Context) {
	listener := http_listener.New(address, p.handler)
	listener.ListenWithCancel(errChan, ctx)
}
