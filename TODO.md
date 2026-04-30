# TODO

A file to track all the things that need to be done in the project.

A '(?)' at the end of an element indicates a possible future feature or idea.

## Proxy
- [x] Add focused `Range` behavior tests for valid ranges, invalid ranges, retry-on-invalid-range, and retry-on-416.
- [x] Expand cache policy matrix tests for methods, statuses, auth/cookies, `no-store`, `private`, `Vary`, and content encoding.
- [x] Add revalidation/stale-cache behavior tests for expired cache hits, upstream failures, and metadata updates.
- [x] Add CONNECT/raw responder regression tests for repeated tunnel requests.
- [x] Add runtime shutdown integration tests for proxy, webserver, and session GC cancellation.
- [x] Introduce a dedicated cache policy component that handles cacheability, `Cache-Control`, `Vary`, auth/cookie headers, content encoding, request methods, and response status.
- [x] Add explicit network timeouts for proxy listeners and upstream HTTP clients.
- [x] Strengthen HTTP protocol boundaries so response/header state cannot leak across CONNECT tunnel requests.
- [x] Make cache keys and stored metadata safe for variant and private responses.
- [x] Fix file and memory cache overwrite accounting so replacing a cached key updates size and entry metrics correctly.
- [x] Fix direct upstream byte accounting for non-cached responses.
- [ ] Add optional file-cache metadata sidecars for restart continuity without turning the cache into long-lived storage. (?)

## Dashboard 


## Project Wide
- [x] Add explicit lifecycle ownership for proxy, webserver, cache janitor, session GC, metrics collectors, and database handles. Current long-lived services are owned by the app lifecycle; add future collectors/DB close hooks there.
- [x] Use `context`/`errgroup`-style startup and shutdown orchestration with bounded graceful shutdown.
- [x] Replace or harden custom concurrency primitives such as `SyncMap` and event subscriptions.
- [x] Make config updates transactional: validate staged changes before committing or firing runtime subscribers.
- [x] Treat security defaults as part of the architecture, including first-login setup, session handling, and conservative cache behavior.
- [ ] Extract and test log SSE tailing for rotation, truncation, partial lines, and transient read/stat failures.
- [ ] Add API JSON/error response helpers and robust request content-type parsing.
- [ ] Encapsulate session state and garbage collection in a runtime-owned session manager.
