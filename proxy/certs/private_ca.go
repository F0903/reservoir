package certs

import (
	"apt_cacher_go/utils/syncmap"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"time"
)

type PrivateCA struct {
	key   crypto.PrivateKey
	cert  *x509.Certificate
	certs *syncmap.SyncMap[string, *tls.Certificate]
}

func NewPrivateCA(certFile, keyFile string) (*PrivateCA, error) {
	caCert, caKey, err := loadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate and key: %v", err)
	}

	if !caCert.IsCA {
		return nil, fmt.Errorf("loaded certificate is not a CA certificate")
	}

	log.Printf("Loaded CA certificate: '%v'\n", caCert.Subject.CommonName)

	return &PrivateCA{
		key:   caKey,
		cert:  caCert,
		certs: syncmap.NewSyncMap[string, *tls.Certificate](),
	}, nil
}

// createCert creates a new certificate/private key pair for the given domains,
// signed by the parent/parentKey certificate. hoursValid is the duration of
// the new certificate's validity.
// https://github.com/eliben/code-for-blog/blob/main/2022/go-and-proxies/connect-mitm-proxy.go
func (ca *PrivateCA) createCert(dnsNames []string, hoursValid int) (cert []byte, priv []byte, err error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"apt-cacher-go"},
		},
		DNSNames:  dnsNames,
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Duration(hoursValid) * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, ca.cert, &privateKey.PublicKey, ca.key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %v", err)
	}
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemCert == nil {
		return nil, nil, errors.New("failed to encode certificate to PEM")
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal private key: %v", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if pemKey == nil {
		return nil, nil, errors.New("failed to encode key to PEM")
	}

	return pemCert, pemKey, nil
}

func (ca *PrivateCA) GetCertForHost(host string) (*tls.Certificate, error) {
	host, _, err := net.SplitHostPort(host)
	if err != nil {
		err := fmt.Errorf("invalid host:port format %v: %v", host, err)
		return nil, err
	}

	if cert, ok := ca.certs.Get(host); ok {
		expired := cert.Leaf.NotAfter.Before(time.Now())
		if expired {
			log.Printf("Certificate for %v is expired, deleting...", host)
			ca.certs.Delete(host)
		} else {
			log.Printf("Using cached certificate for %v", host)
			return cert, nil
		}
	}

	log.Printf("Creating new certificate for %v", host)

	// Create a fake TLS certificate for the target host, signed by our CA.
	pemCert, pemKey, err := ca.createCert([]string{host}, 240)
	if err != nil {
		err := fmt.Errorf("failed to create TLS certificate for %v: %v", host, err)
		return nil, err
	}

	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		err := fmt.Errorf("failed to create X509 key pair for cert %v: %v", tlsCert, err)
		return nil, err
	}

	log.Printf("Created certificate for %v", host)

	ca.certs.Set(host, &tlsCert)

	return &tlsCert, nil
}
