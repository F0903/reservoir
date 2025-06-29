package certs

import (
	"crypto/tls"
)

type CertAuthority interface {
	GetCertForHost(host string) (tls.Certificate, error)
}
