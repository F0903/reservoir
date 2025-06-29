package certs

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// loadX509KeyPair loads a certificate/key pair from files, and unmarshals them
// into data structures from the x509 package. Note that private key types in Go
// don't have a shared named interface and use `any` (for backwards
// compatibility reasons).
// https://github.com/eliben/code-for-blog/blob/main/2022/go-and-proxies/connect-mitm-proxy.go
func loadX509KeyPair(certFile, keyFile string) (cert *x509.Certificate, key any, err error) {
	cf, err := os.ReadFile(certFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate file %s: %v", certFile, err)
	}

	kf, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key file %s: %v", keyFile, err)
	}
	certBlock, _ := pem.Decode(cf)
	cert, err = x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	keyBlock, _ := pem.Decode(kf)
	key, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return cert, key, nil
}
