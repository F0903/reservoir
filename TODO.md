# TODO

A file to track all the things that need to be done in the project.

A '(?)' at the end of an element indicates a possible future feature or idea.

## Proxy
- Metric for request latency
- Rewrite caching to store in memory, and evict to disk when necessary
- More tests
- Content-Type Specific Handling (?)

## Dashboard
- Widget for request latency metric
- More performance metrics
- Log coloring / better readability
- Mobile support

### Current Tasks:

#### Metrics Refactor
- [ ] Enhance Request Metrics: Break down total requests by status code (2xx, 4xx, 5xx).
- [ ] Introduce Latency Tracking: Measure request duration for cache hits vs. misses.
- [ ] Improve Data Transfer Insights: Track bytes fetched from upstream to calculate bandwidth savings.
