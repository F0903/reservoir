package proxy

import (
	"context"
	"reservoir/proxy/certs"
	"reservoir/utils/httplistener"
)

type Proxy struct {
	handler *proxyHandler
}

// Creates a new MITM proxy. It should be passed the filenames
// for the certificate and private key of a certificate authority trusted by the
// client's machine.
func New(cacheDir string, ca certs.CertAuthority, ctx context.Context) (*Proxy, error) {
	handler, err := newCachingMitmProxyHandler(cacheDir, ca, ctx)
	if err != nil {
		return nil, err
	}
	return &Proxy{
		handler: handler,
	}, nil
}

func (p *Proxy) Listen(address string, errChan chan error, ctx context.Context) {
	listener := httplistener.New(address, p.handler)
	listener.ListenWithCancel(errChan, ctx)
}
