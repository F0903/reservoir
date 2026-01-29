package tests

import (
	"crypto/tls"
	"fmt"
)

// FakeCA satisfies certs.CertAuthority for testing purposes where we don't intercept HTTPS.
type FakeCA struct{}

func (c *FakeCA) GetCertForHost(host string) (*tls.Certificate, error) {
	return nil, fmt.Errorf("not implemented in FakeCA")
}
