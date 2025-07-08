package proxy

import (
	"apt_cacher_go/proxy/certs"
	"log"
	"net/http"
)

type CachingMitmProxy struct {
	handler *cachingMitmProxyHandler
}

// Creates a new MITM proxy. It should be passed the filenames
// for the certificate and private key of a certificate authority trusted by the
// client's machine.
func NewCachingMitmProxy(cacheDir string, ca certs.CertAuthority) (*CachingMitmProxy, error) {
	handler, err := newCachingMitmProxyHandler(cacheDir, ca)
	if err != nil {
		return nil, err
	}
	return &CachingMitmProxy{
		handler: handler,
	}, nil
}

func (p *CachingMitmProxy) ListenBlocking(address string) error {
	return http.ListenAndServe(address, p.handler)
}

func (p *CachingMitmProxy) Listen(address string) {
	go func() {
		if err := p.ListenBlocking(address); err != nil {
			log.Println("Error during non-blocking listen:", err)
		}
	}()
}
