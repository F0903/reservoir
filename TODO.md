# TODO

A file to track all the things that need to be done in the project.

A '(?)' at the end of an element indicates a possible future feature or idea.

## Proxy

## Dashboard 
- [ ] Make widgets more responsive depending on their size.

## Project Wide
- [ ] Move runtime-owned paths such as config, database, cache, logs, and certs out of package globals and into injected runtime/app configuration.
- [ ] Make config PATCH handling reject unknown keys and move update/persist/restart tracking behind a synchronized config service.
- [ ] Decouple proxy/cache packages from global metrics and concrete cache backend construction by injecting metrics and cache factories.
- [ ] Split dashboard-specific CSP/assets from generic webserver hardening so API-only or dashboard-disabled builds remain clean.
