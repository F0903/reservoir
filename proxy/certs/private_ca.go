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
	"log/slog"
	"math/big"
	"net"
	"time"
)

var (
	ErrFailedToEncodeCert     = errors.New("failed to encode certificate to PEM")
	ErrFailedToEncodeKey      = errors.New("failed to encode key to PEM")
	ErrFailedToLoadCA         = errors.New("failed to load CA certificate and key")
	ErrNotCACertificate       = errors.New("loaded certificate is not a CA certificate")
	ErrFailedToGenerateKey    = errors.New("failed to generate private key")
	ErrFailedToGenerateSerial = errors.New("failed to generate serial number")
	ErrFailedToCreateCert     = errors.New("failed to create certificate")
	ErrFailedToMarshalKey     = errors.New("failed to marshal private key")
	ErrInvalidHostPort        = errors.New("invalid host:port format")
	ErrFailedToCreateTLSCert  = errors.New("failed to create TLS certificate for host")
	ErrFailedToCreateX509Pair = errors.New("failed to create X509 key pair for cert")
)

type PrivateCA struct {
	key   crypto.PrivateKey
	cert  *x509.Certificate
	certs *syncmap.SyncMap[string, *tls.Certificate]
}

func NewPrivateCA(certFile, keyFile string) (*PrivateCA, error) {
	caCert, caKey, err := loadX509KeyPair(certFile, keyFile)
	if err != nil {
		slog.Error("Failed to load CA certificate and key", "cert_file", certFile, "key_file", keyFile, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrFailedToLoadCA, err)
	}

	if !caCert.IsCA {
		slog.Error("Loaded certificate is not a CA certificate", "cert_file", certFile)
		return nil, ErrNotCACertificate
	}

	slog.Info("Loaded CA certificate", "common_name", caCert.Subject.CommonName)

	return &PrivateCA{
		key:   caKey,
		cert:  caCert,
		certs: syncmap.New[string, *tls.Certificate](),
	}, nil
}

// createCert creates a new certificate/private key pair for the given domains,
// signed by the parent/parentKey certificate. hoursValid is the duration of
// the new certificate's validity.
// https://github.com/eliben/code-for-blog/blob/main/2022/go-and-proxies/connect-mitm-proxy.go
func (ca *PrivateCA) createCert(dnsNames []string, hoursValid int) (cert []byte, priv []byte, err error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		slog.Error("Failed to generate private key", "error", err)
		return nil, nil, fmt.Errorf("%w: %v", ErrFailedToGenerateKey, err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		slog.Error("Failed to generate serial number", "error", err)
		return nil, nil, fmt.Errorf("%w: %v", ErrFailedToGenerateSerial, err)
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
		slog.Error("Failed to create certificate", "error", err)
		return nil, nil, fmt.Errorf("%w: %v", ErrFailedToCreateCert, err)
	}
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemCert == nil {
		slog.Error("Failed to encode certificate to PEM")
		return nil, nil, ErrFailedToEncodeCert
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		slog.Error("Failed to marshal private key", "error", err)
		return nil, nil, fmt.Errorf("%w: %v", ErrFailedToMarshalKey, err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if pemKey == nil {
		slog.Error("Failed to encode key to PEM")
		return nil, nil, ErrFailedToEncodeKey
	}

	return pemCert, pemKey, nil
}

func (ca *PrivateCA) GetCertForHost(host string) (*tls.Certificate, error) {
	host, _, err := net.SplitHostPort(host)
	if err != nil {
		slog.Error("Invalid host:port format", "host", host, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrInvalidHostPort, err)
	}

	if cert, ok := ca.certs.Get(host); ok {
		expired := cert.Leaf.NotAfter.Before(time.Now())
		if expired {
			slog.Warn("Certificate for %v is expired, deleting...", "host", host)
			ca.certs.Delete(host)
		} else {
			slog.Debug("Using cached TLS certificate", "host", host, "expires", cert.Leaf.NotAfter)
			return cert, nil
		}
	}

	slog.Debug("Creating new TLS certificate", "host", host)

	// Create a fake TLS certificate for the target host, signed by our CA.
	pemCert, pemKey, err := ca.createCert([]string{host}, 240)
	if err != nil {
		slog.Error("Failed to create TLS certificate for host", "host", host, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrFailedToCreateTLSCert, err)
	}

	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		slog.Error("Failed to create X509 key pair for cert", "host", host, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrFailedToCreateX509Pair, err)
	}

	slog.Info("Created TLS certificate", "host", host, "expires", tlsCert.Leaf.NotAfter)

	ca.certs.Set(host, &tlsCert)

	return &tlsCert, nil
}
