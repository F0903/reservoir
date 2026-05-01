# reservoir

[![CI](https://github.com/F0903/reservoir/actions/workflows/ci.yml/badge.svg)](https://github.com/F0903/reservoir/actions/workflows/ci.yml)

A caching and coalescing MITM (Man-in-the-Middle) HTTP(S) forward proxy with an embedded dashboard.

Supports caching of both HTTP and HTTPS requests by injecting a certificate to decrypt and cache the data before sending it back to the client.

The original intended usage is as a central cache proxy for apt or other package managers when spread across multiple containers.

The dashboard is directly embedded into the executable, so the final build artifact is a single self-contained executable.

## Requirements

- Go 1.26 or newer
- OpenSSL (for generating CA cert/key)

## Usage Guide

To start with, you need to generate a CA certificate and key to be used as your own makeshift certificate authority to generate new certificates for requests to HTTPS domains. This is the mechanism that allows the proxy to decrypt and cache HTTPS responses, with the caveat being that EVERY client that proxies HTTPS requests through this MUST trust this CA certificate, otherwise you will get errors relating to the untrusted cert.

### Generate a CA Certificate and Key

This generates a CA certificate with the common name (CN) "reservoir" and corresponding key, both in PEM format.

```sh
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj "//CN=reservoir"
```

### Trust the CA Certificate

- **Linux:** Add to `/usr/local/share/ca-certificates/` and run `sudo update-ca-certificates`.
- **Windows:** Double-click `ca.crt`, click "Install Certificate..." and follow the wizard.

## Running the Proxy

To run the proxy you have two options; you can either download a prebuilt binary from the releases page (or an unstable build from the actions page), or you can build it yourself by following the instructions below:

### Building the Proxy

Before getting started, you will need to install a few dependencies:

#### Installing dependencies

First off you will need to have [**Node.js** installed](https://nodejs.org/en/download) and **pnpm** enabled *(run the command `corepack enable pnpm`)*.

Then you need to have **GNU make** installed.
The way to do this will vary depending on your OS. If you use a Linux distro it will be easy to install via your package manager (it might even be preinstalled).

On Windows the easiest way to do this is with [Chocolatey](https://chocolatey.org/install#individual) by running `choco install make`.  
Alternatively you can [install it manually here](https://gnuwin32.sourceforge.net/packages/make.htm).

You will of course also need to have [Go installed](https://go.dev/dl/).

Then you just have to build the project with **make** by running `make` in the project directory.

This will automatically build the frontend and the final proxy executable.

The resulting executable is now ready in the project root to be copied standalone to wherever you wish.   
If you are running it on Linux, you can also setup a systemd service for it, which is recommended.

## Proxy Configuration

Configuration can be done either via the generated configuration file, or the command-line arguments.

If a setting is both specified in the configuration file and as a command-line argument, the command-line argument will take precedence.

### Configuration File

The configuration file is a JSON file that contains all the settings for the proxy.
You can edit this file manually in ``var/config.json`` to change the configuration. If the ``var/`` folder or config does not exist, run the proxy once, and it will be created automatically.
Some settings can also be changed in the Dashboard.

### Dashboard Bootstrap Login

When the API and dashboard are enabled and the user database is empty, Reservoir starts in first-run bootstrap mode.

The first admin username can be chosen during setup. The bootstrap password must be at least 12 characters, is never written to disk, and the created user is signed in immediately after setup. Once a user exists, the bootstrap endpoint returns a conflict and normal login is required.

Older installs that still have the legacy generated bootstrap admin may continue to receive a generated password in `var/bootstrap-admin-password.txt` until that account changes its password. New empty installs use the first-run bootstrap page instead.

If the API is disabled, Reservoir cannot create dashboard users or accept dashboard logins.

### Package Cache Behavior

Reservoir is tuned first as a shared package cache for package-manager traffic. By default the proxy cache policy favors useful package caching over strict upstream cache directives:

- `proxy.cache_policy.ignore_cache_control` defaults to `true`, so package responses can still be cached when upstream sends directives such as `no-store`.
- `proxy.cache_policy.force_default_max_age` defaults to `true`, so cached responses use `proxy.cache_policy.default_max_age` instead of upstream freshness metadata.
- Requests containing `Authorization` or `Cookie` are not stored in the shared cache.
- Responses with `Set-Cookie`, unsupported `Vary`, or unsafe content encoding metadata are not stored in the shared cache.
- When a cached package response is stale and upstream revalidation fails with a server error or network failure, Reservoir serves the stale cached response.

These defaults are intentional for package-cache deployments. If you need stricter general-purpose proxy semantics, disable `ignore_cache_control` and `force_default_max_age` in `var/config.json`.

### Cache Backends

Reservoir supports two cache backends:

- `cache.type = "memory"` keeps cached responses in process memory. This is the default and is best for short-lived burst caching where the proxy only needs to coalesce many package-manager requests that happen close together.
- `cache.type = "file"` stores cached response bodies under `cache.file.dir`. This is useful when cached package responses may be larger than the memory budget or when short restart continuity is useful.

The file cache writes metadata sidecars next to cached response bodies. On startup, Reservoir loads sidecars only when the matching cached body still exists, the body is non-empty, and the cached response has not expired. Expired, corrupt, or orphaned cache files are discarded. This preserves useful restart continuity without turning the proxy into a long-lived package repository.

Loaded file-cache entries are still subject to `cache.max_cache_size`, normal expiry, and the cleanup interval. If restored entries exceed the configured cache size, startup eviction trims them before serving traffic.

### Command-Line Arguments

You can always display info about the command-line arguments by running the proxy with the `--help` flag. Command-line arguments only override the generated configuration when they are supplied.

The command-line arguments currently available are the following:

- **version** - Print the Reservoir version and exit.
- **listen** (:9999) - The address and port that the proxy will listen on.
- **ca-cert** (ssl/ca.crt) - The path to the PEM cert of the CA the proxy will use to sign.
- **ca-key** (ssl/ca.key) - The path to the PEM key of the CA the proxy will use to sign.
- **cache-dir** (var/cache/) - The path where the file cache should be stored.
- **webserver-listen** (localhost:8080) - The address and port that the webserver (dashboard and API) will listen on.
- **no-dashboard** (false) - Disable the embedded dashboard.
- **no-api** (false) - Disable the API.
- **log-level** (info) - Set the logging level (DEBUG, INFO, WARN, ERROR).
- **log-file** (var/proxy.log) - The path to the log file. Setting this to an empty value disables file logging and the dashboard log viewer.
- **log-file-max-size** (500M) - The maximum size of the log file before it is rotated.
- **log-file-max-backups** (3) - The maximum number of old log files to keep.
- **log-file-compress** (true) - Enable compression for rotated log files.
- **log-to-stdout** (false) - Enable logging to stdout.

Other cache settings are currently configured through `var/config.json` or the dashboard rather than command-line flags. The most important ones are:

- `cache.type` - `memory` or `file`.
- `cache.max_cache_size` - Maximum total cache size.
- `cache.cleanup_interval` - How often expired entries and over-budget cache data are cleaned up.
- `cache.memory.memory_budget_percent` - Memory-cache budget as a percentage of total system memory.
- `proxy.cache_policy.ignore_cache_control` - Whether to ignore upstream cache-control directives.
- `proxy.cache_policy.force_default_max_age` - Whether to always use Reservoir's configured default freshness lifetime.
- `proxy.cache_policy.default_max_age` - The fallback/default freshness lifetime for cached responses.

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
