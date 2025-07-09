# apt-cacher-go

A simple caching MITM forward proxy with a dashboard, written in Go and Svelte.

Supports caching of both HTTP and HTTPS requests by injecting its own certificate to decrypt and cache the data before sending it back to the client.

The prime usage of this is as a central cache proxy for apt.

The dashboard is directly embedded into the executable, so the final build artifact is a single file.

[Based on the MITM forward proxy example written by Eli Bendersky](https://github.com/eliben/code-for-blog/blob/main/2022/go-and-proxies/connect-mitm-proxy.go)

## Requirements

- Go 1.24 or newer
- OpenSSL (for generating CA cert/key)

## Usage Guide

To start with, you need to generate a certificate and key to be used as a certificate authority to generate new certificates for requests to HTTPS domains. This is the mechanism that allows the proxy to decrypt and cache HTTPS responses.
The caveat being that EVERY client that proxies HTTPS requests through this MUST trust this CA certificate, otherwise you will get errors.

### Generate a CA Certificate and Key (PEM format)

```sh
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj "//CN=apt-cacher-go"
```

### Trust the CA Certificate

- **Linux:** Add to `/usr/local/share/ca-certificates/` and run `sudo update-ca-certificates`.
- **Windows:** Double-click `ca-cert.pem` and install it to "Trusted Root Certification Authorities".
- **macOS:** Use Keychain Access to import and trust the certificate.

### Running the Proxy

First you have to build the executable with make by running the following command in the project directory:

```sh
make build
```

This will automatically build both the frontend and the proxy executable.

Then simply copy the resulting executable to whereever you wish, and run as normal. If you are running it on Linux, you can also setup a systemd service for it.

### Note When Updating

When updating it is recommended to delete the local var/ folder, as changes to the config or metadata format could cause unexpected behaviour.

## Proxy Configuration

Configuration currently takes place via command-line arguments.

The arguments currently available are the following:

- **listen** (0.0.0.0:9999) - The address and port that the proxy will listen on.
- **ca-cert** (ssl/ca.crt) - The path to the PEM cert of the CA the proxy will use to sign.
- **ca-key** (ssl/ca.key) - The path to the PEM key of the CA the proxy will use to sign.
- **cache-dir** (var/cache) - The path where the cache should be stored.
- **dashboard-listen** (localhost:8080) - The address and port that the dashboard will listen on.

## Example: Using curl with the Proxy

```sh
curl -x http://127.0.0.1:9999 https://example.com/
```

If your CA is not trusted by the system, you can specify it for curl:

```sh
curl --cacert ca-cert.pem -x http://127.0.0.1:9999 https://example.com/
```

Alternatively if you are too lazy to specify the cert (like me), you can use the `-k` option to skip cert validation:

```sh
curl -k -x http://127.0.0.1:9999 https://example.com/
```
