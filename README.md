# MITM Forward Proxy

A simple MITM forward proxy written in Go.

[Based on the one written by Eli Bendersky](https://github.com/eliben/code-for-blog/blob/main/2022/go-and-proxies/connect-mitm-proxy.go)

## Requirements

- Go 1.24.4 or newer (older versions will most likely work but are untested)
- OpenSSL (for generating CA cert/key)

## Quick Start

### Generate a CA Certificate and Key (PEM format)

```sh
openssl genrsa -out ca-key.pem 2048
openssl req -x509 -new -nodes -key ca-key.pem -sha256 -days 3650 -out ca-cert.pem -subj "/CN=MITMProxy"
```

### Trust the CA Certificate

- **Windows:** Double-click `ca-cert.pem` and install it to "Trusted Root Certification Authorities".
- **macOS:** Use Keychain Access to import and trust the certificate.
- **Linux:** Add to `/usr/local/share/ca-certificates/` and run `sudo update-ca-certificates`.
- **Browsers:** You may need to import the CA in your browser's settings for HTTPS interception.

### Run the Proxy

```sh
go run main.go -address 127.0.0.1:9999 -ca-cert ca-cert.pem -ca-key ca-key.pem
```

Available arguments:

- **address** - The address and port that the proxy will listen on.
- **ca-cert** - The path to the PEM cert of the CA the proxy will use to sign.
- **ca-key** - The path to the PEM key of the CA the proxy will use to sign.

## Example: Using curl with the Proxy

```sh
curl -x http://127.0.0.1:9999 https://example.com/
```

If your CA is not trusted by the system, you can specify it for curl:

```sh
curl --cacert ca-cert.pem -x http://127.0.0.1:9999 https://example.com/
```
