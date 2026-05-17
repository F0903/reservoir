# TODO

A file to track all the things that need to be done in the project.

A '(?)' at the end of an element indicates a possible future feature or idea.

## Proxy
- [x] Add "hybrid" cache backend that combines file and in-memory caching.
- [x] Optimize hybrid cache writes to avoid redundant buffering and stream oversized entries to file.

## Dashboard 
- [ ] Make widgets more responsive depending on their size. 
- [x] Use custom tooltip component more places instead of browser tooltips.

## Project Wide
- [ ] Move runtime-owned paths such as config, database, cache, logs, and certs out of package globals and into injected runtime/app configuration.
- [x] Make user/admin store mutations transactionally protect invariants, especially preventing concurrent removal of the last admin.
- [x] Centralize database/store lifecycle ownership in runtime instead of opening concrete stores from API/auth request handling.
- [ ] Introduce a small application/service layer so API endpoint context does not need to depend directly on config, cache, DB stores, PHC, and auth internals.
- [ ] Make config PATCH handling reject unknown keys and move update/persist/restart tracking behind a synchronized config service.
- [ ] Decouple proxy/cache packages from global metrics and concrete cache backend construction by injecting metrics and cache factories.
- [ ] Split dashboard-specific CSP/assets from generic webserver hardening so API-only or dashboard-disabled builds remain clean.
