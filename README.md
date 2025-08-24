# reservoir

A caching MITM (Man-in-the-Middle) forward proxy written in Go with an embedded dashboard written with Svelte.

Supports caching of both HTTP and HTTPS requests by injecting its own certificate to decrypt and cache the data before sending it back to the client.

The original intended usage is as a central cache proxy for apt or other package managers.

The dashboard is directly embedded into the executable, so the final build artifact is a single file.

## Requirements

- Go 1.24 or newer
- OpenSSL (for generating CA cert/key)

## Usage Guide

To start with, you need to generate a certificate and key to be used as a certificate authority to generate new certificates for requests to HTTPS domains. This is the mechanism that allows the proxy to decrypt and cache HTTPS responses.
The caveat being that EVERY client that proxies HTTPS requests through this MUST trust this CA certificate, otherwise you will get errors relating to the untrusted cert.

### Generate a CA Certificate and Key (PEM format)

```sh
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj "//CN=reservoir"
```

### Trust the CA Certificate

- **Linux:** Add to `/usr/local/share/ca-certificates/` and run `sudo update-ca-certificates`.
- **Windows:** Double-click `ca-cert.pem` and install it to "Trusted Root Certification Authorities".
- **macOS:** Use Keychain Access to import and trust the certificate.

### Running the Proxy

Before getting started, you will need to install a few dependencies:

#### Installing dependencies

First off you will need to have [**Node** installed](https://nodejs.org/en/download) and **pnpm** enabled *(run the command `corepack enable pnpm`)*.

Then you need to have **GNU make** installed.
The way to do this will vary depending on your OS. If you run a Linux distro it will be easy to install via your package manager (it might even be preinstalled).

On Windows the easiest way to do this is with [Chocolatey](https://chocolatey.org/install#individual) by running `choco install make`. Alternatively you can [install it manually here](https://gnuwin32.sourceforge.net/packages/make.htm).

You will of course also need to have [Go installed](https://go.dev/dl/).

Then you just have to build the project with **make** by running `make` in the project directory.

This will automatically build both the frontend and the proxy executable.

Then simply copy the resulting executable to whereever you wish, and run as normal. If you are running it on Linux, you can also setup a systemd service for it, which is recommended.

### Note When Updating

When updating it is recommended to delete the local ``var/cache/`` folder and ``var/config.json``, as changes to the config or metadata format could cause unexpected behaviour.

## Proxy Configuration

Configuration can be done either via the generated configuration file, or the command-line arguments.

If a setting is both specified in the configuration file and as a command-line argument, the command-line argument will take precedence.

### Configuration File

The configuration file is a JSON file that contains all the settings for the proxy.
You can edit this file manually in ``var/config.json`` to change the configuration. If the ``var/`` folder or config does not exist, run the proxy once, and it will be created automatically.
Some settings can also be changed in the Dashboard.

### Command-Line Arguments

You can always display info about the command-line arguments by running the proxy with the `--help` flag. Otherwise, you can refer to the following list.

The command-line arguments currently available are the following:

- **listen** (0.0.0.0:9999) - The address and port that the proxy will listen on.
- **ca-cert** (ssl/ca.crt) - The path to the PEM cert of the CA the proxy will use to sign.
- **ca-key** (ssl/ca.key) - The path to the PEM key of the CA the proxy will use to sign.
- **cache-dir** (var/cache) - The path where the cache should be stored.
- **webserver-listen** (localhost:8080) - The address and port that the webserver (dashboard and API) will listen on.
- **no-dashboard** (false) - Disable the embedded dashboard.
- **no-api** (false) - Disable the API.
- **log-level** (info) - Set the logging level (DEBUG, INFO, WARN, ERROR).
- **log-file** (var/proxy.log) - The path to the log file. If no path is specified, no file logging will be done. (will also disable dashboard log-viewer)
- **log-file-max-size** (500M) - The maximum size of the log file before it is rotated.
- **log-file-max-backups** (3) - The maximum number of old log files to keep.
- **log-file-compress** (true) - Enable compression for rotated log files.
- **log-to-stdout** (false) - Enable logging to stdout.

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
