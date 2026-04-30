# TODO

A file to track all the things that need to be done in the project.

A '(?)' at the end of an element indicates a possible future feature or idea.

## Proxy
- [ ] More tests. Started: added regression and integration coverage for cache policy, variants, config updates, ranges, and responders.
- [x] Introduce a dedicated cache policy component that handles cacheability, `Cache-Control`, `Vary`, auth/cookie headers, content encoding, request methods, and response status.
- [x] Add explicit network timeouts for proxy listeners and upstream HTTP clients.
- [x] Strengthen HTTP protocol boundaries so response/header state cannot leak across CONNECT tunnel requests.
- [x] Make cache keys and stored metadata safe for variant and private responses.

## Dashboard 


## Project Wide
- [x] Add explicit lifecycle ownership for proxy, webserver, cache janitor, session GC, metrics collectors, and database handles. Current long-lived services are owned by the app lifecycle; add future collectors/DB close hooks there.
- [x] Use `context`/`errgroup`-style startup and shutdown orchestration with bounded graceful shutdown.
- [ ] Replace or harden custom concurrency primitives such as `SyncMap` and event subscriptions. Started: `SyncMap` iteration and session access are hardened; event subscriptions remain.
- [x] Make config updates transactional: validate staged changes before committing or firing runtime subscribers.
- [ ] Treat security defaults as part of the architecture, including first-login setup, session handling, and conservative cache behavior. Started: shared-cache and variant safety improved; first-login/session defaults remain.
