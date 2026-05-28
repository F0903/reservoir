# TODO

A file to track all the things that need to be done in the project.

A '(?)' at the end of an element indicates a possible future feature or idea.

## Proxy
- [x] Fix HTTPS CONNECT cache-key scheme handling so HTTP and HTTPS entries cannot collide.
- [x] Enforce hard memory-cache read/write limits for oversized upstream responses.
- [x] Add bounded eviction/cleanup for generated per-host MITM certificates.
- [x] Validate `cache.lock_shards` is greater than zero before cache construction.
- [x] Revisit invalid config load behavior so a typo does not silently reset tuned config.

## Dashboard 
- [ ] Make widgets more responsive depending on their size.
- [ ] Set session cookies `Secure` when the dashboard/API are served over HTTPS.

## Project Wide
- [x] Upgrade vulnerable Go toolchain/module versions flagged by `govulncheck`.
- [x] Upgrade vulnerable Svelte/SvelteKit/frontend lockfile dependencies flagged by `pnpm audit`.
- [x] Reuse managed-password validation for self-service password changes.
- [x] Close old file loggers when logging configuration is hot-swapped.
