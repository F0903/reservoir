# TODO

A file to track all the things that need to be done in the project.

A '(?)' at the end of an element indicates a possible future feature or idea.

## Must Fix

Highest priority first.

- [x] Memory cache deadlocks when full: eviction waits on a shard lock the write already holds. Use `TryRLock` and skip busy entries. (`cache/memory/cache.go:74`)
- [x] One client disconnecting fails the shared download for all waiters, and a stalled upstream body hangs that URL forever. Detach the download from the client's context and add a read-idle timeout. (`proxy/fetch/fetcher.go:70`)
- [x] A failed cache write (empty body with the file cache, full disk) returns 502. Wrap the error as `errNotCacheable` so the direct-fetch fallback runs. (`proxy/fetch/upstream.go:85`)
- [x] Upstream scheme comes from `upstream_default_https`, not the request: `true` breaks HTTP-only hosts, `false` sends HTTPS requests unencrypted. Use https when `req.TLS != nil`, otherwise the client's scheme. Ship together with the next item. (`proxy/fetch/requests.go:71`)
- [x] The proxy is open on all interfaces with no blocked destinations, so it can reach internal services and, once the scheme is fixed, the dashboard bootstrap on localhost. Refuse loopback and link-local addresses in a `net.Dialer.Control` hook. (`config/proxy_config.go:46`)
- [x] A failed response inside the HTTPS tunnel leaves the connection open and the client hangs. `break` out of the loop on error. Ship together with the next item, or HEAD responses with a known size will close the connection. (`proxy/connect.go:80`)
- [x] 204, 304 and size-less HEAD responses in the tunnel get chunked framing and a stray `0\r\n\r\n`. Skip chunking for bodyless statuses and set `response.Request`. (`proxy/responder/raw_http_responder.go:78`)
- [x] Encoded paths are decoded before going upstream (`%2F` becomes `/`), and different URLs share a cache key (`/dir` and `/dir/`). Copy `RawPath` and build the key from `EscapedPath()` without `path.Clean`. (`proxy/fetch/requests.go:94`, `cache/cache_key.go:37`)

## Should Fix

Highest priority first.

- [x] Session IDs are written to debug logs that non-admins can read. Remove `session_id` from the six log lines and make the log endpoints admin-only unless non-admins need them. (`webserver/auth/session.go:112`, `webserver/api/endpoints/log/log.go:24`)
- [x] Private responses can be served to other clients: `private` is ignored by default and custom token headers aren't treated as credentials. Always honour `private` and add common token headers (`Private-Token`, `X-Api-Key`). (`proxy/fetch/cache_policy.go:62`, `proxy/headers/header_directives.go:136`)
- [x] The login endpoint can exhaust the server: no body limit, random usernames bypass the limiter and cost 64 MB of hashing each, the attempts map never shrinks, and users behind a reverse proxy share one IP. Add `http.MaxBytesReader`, limit concurrent hashing, also limit per IP, and sweep old records. (`webserver/api/apihttp/response.go:66`, `webserver/auth/login_limiter.go:17`)
- [x] A slow download blocks unrelated cached files on the same shard because the body is read while holding the lock. Read first, lock only for the rename and map update. (`cache/file/operations.go:69`, `cache/memory/operations.go:132`, `cache/hybrid/write.go:63`)
- [x] Memory use can pass the budget: it uses host RAM inside containers, ignores downloads in progress, and promotion reads the whole file before checking for room. Cap with `/sys/fs/cgroup/memory.max`, stream large files to disk, and check room first. (`cache/memory/cache.go:41`, `cache/hybrid/write.go:149`, `cache/hybrid/promotion.go:21`)
- [x] A crash leaves `*.tmp` files forever, mismatched body and metadata files, or an empty config that blocks startup. Delete `*.tmp` at startup, discard entries whose size doesn't match, and save config via temp file and rename. (`cache/file/metadata.go:50`, `cache/file/metadata.go:90`, `config/config.go:67`)
- [x] Decrypted tunnel connections have no time limit. Call `SetDeadline` before the handshake and before each `ReadRequest`. (`proxy/connect.go:54`, `proxy/connect.go:67`)
- [x] HTTPS requests never reuse upstream connections on HTTP/1.1. Delete `req.Close = true`. (`proxy/connect.go:79`)
- [x] Range requests always go upstream even when the file is cached, and ranges past the end of a file get 416 instead of being trimmed. `tests/http_test.go:132` expects the upstream hit, so confirm whether that part is deliberate. (`proxy/fetch/fetcher.go:61`, `proxy/headers/range_header.go:55`)
- [x] Strict cache mode refuses `Cache-Control: public` with no `max-age`. Delete the `maxAge < 1` check; the parser already handles `max-age=0`. (`proxy/headers/header_directives.go:145`)
- [x] A failed user update is half saved because the password is validated after the username and role are written. Validate first. (`application/services/users.go:68`)
- [x] Delete unused code, about 200 lines: unknown-size hybrid writes and their tests, unused listener variants, session wrapper functions and four error values. (`cache/hybrid/spill.go:1`, `utils/httplistener/http_listener.go:58`, `webserver/auth/session.go:203`, `proxy/proxy.go:16`)

- [x] Login limits use the direct peer address, so users behind a reverse proxy share them. Needs a trusted-proxy setting before `X-Forwarded-For` can be read safely. (`webserver/auth/login_limiter.go`)
- [x] A client that stops reading during a download holds its connection forever, in both the tunnel and plain HTTP paths. Add a per-write deadline. (`proxy/responder/raw_http_responder.go`, `utils/httplistener/http_listener.go`)
- [x] A user update can still be half saved if a later store write fails, such as renaming and demoting the last admin together. Do the writes in one transaction. (`application/services/users.go`)

## Nice to Have

- [x] Generated certificates use `NotBefore: time.Now()`, so clients with a slightly slow clock reject them. Backdate by an hour. (`proxy/certs/private_ca.go:129`)
- [x] Cached responses with no upstream ETag are sent with an empty `ETag` header. Only set it when there is a value. (`proxy/http.go:103`, `proxy/range.go:72`)
- [x] Shutdown panics if active requests outlast the 30-second timeout. Force-close the server instead of returning the error. (`utils/httplistener/http_listener.go:51`, `main.go:33`)
- [x] The Vary index never shrinks when entries are evicted. Remove entries with the cache entry or cap its size. (`proxy/fetch/variants.go:60`)
- [x] `password_change_required` is ignored when a user update has no password. Apply it on its own. (`application/services/users.go:82`)

## Proxy

## Dashboard 
- [ ] Make widgets more responsive depending on their size.

## Project Wide
