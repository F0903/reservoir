# Copilot Instructions for apt-cacher-go

## Project Overview

This is a caching MITM (Man-in-the-Middle) forward proxy written in Go with a SvelteKit dashboard. It caches both HTTP and HTTPS requests by injecting its own certificate authority to decrypt and cache data before sending it back to the client. The primary use case is as a central cache proxy for apt repositories.

## Architecture & Key Components

### Core Architecture

- **Language**: Go 1.24.4 with modern ServeMux patterns
- **Frontend**: SvelteKit with TypeScript and dark-first theming
- **Build System**: Embedded frontend in Go binary for single-file deployment
- **Caching Strategy**: File-based cache with in-memory metadata optimization
- **Concurrency**: Thread-safe design using atomic operations and custom sync primitives

### Package Structure

```
apt_cacher_go/
├── main.go                 # Application entry point
├── cache/                  # Cache implementation
├── config/                 # Configuration management
├── metrics/                # Thread-safe metrics system
├── proxy/                  # MITM proxy implementation
├── webserver/             # Web server and API
│   ├── api/               # REST API endpoints
│   └── dashboard/         # SvelteKit frontend
└── utils/                 # Utility packages
```

## Core Patterns & Best Practices

### 1. Thread-Safe Metrics System

- **Pattern**: Use atomic operations for all metrics
- **Implementation**: `AtomicInt64` and `AtomicTime` with JSON marshaling
- **Example**:

```go
type AtomicInt64 struct {
    metric int64
}

func (m *AtomicInt64) Increment() {
    atomic.AddInt64(&m.metric, 1)
}

func (m AtomicInt64) MarshalJSON() ([]byte, error) {
    value := atomic.LoadInt64(&m.metric)
    return json.Marshal(value)
}
```

### 2. Cache System Architecture

- **Pattern**: Dual-layer caching with file storage and in-memory metadata
- **Key Components**:
  - `FileCache[ObjectData]`: Generic cache implementation
  - `EntryMetadata[ObjectData]`: Metadata with custom types
  - `CacheKey`: Blake2b-based cache keys from HTTP requests
- **Eviction Strategy**: Hybrid LRU + size-based priority scoring
- **Thread Safety**: Per-key locking with `sync.RWMutex`

### 3. Configuration Management

- **Pattern**: JSON-based configuration with version management
- **Implementation**: `LoadOrDefault` pattern with persistence
- **Custom Types**: `bytesize.ByteSize` and `duration.Duration` with JSON marshaling
- **Global Access**: Single global config instance with lazy initialization

### 4. Custom Type System

- **ByteSize**: Human-readable byte size parsing (e.g., "1G", "512M")
- **Duration**: Extended duration parsing with custom format
- **Optional**: Generic optional type implementation
- **All custom types implement JSON marshaling/unmarshaling**

### 5. HTTP Proxy Architecture

- **MITM Handling**: Certificate authority injection for HTTPS
- **Cache Directives**: HTTP cache control parsing and validation
- **Conditional Requests**: Support for If-Modified-Since, ETag, etc.
- **Metrics Integration**: Request counting and byte tracking

## Development Guidelines

### Code Style

- **Error Handling**: Always wrap errors with context using `fmt.Errorf`
- **Logging**: Use structured logging with request context
- **Locking**: Prefer read locks where possible, use TryLock for eviction
- **Memory Management**: Close resources in defer statements

### Testing Patterns

- **File Operations**: Use `asserted_path` package for safe path handling
- **Concurrent Code**: Test with race detector enabled
- **Cache Testing**: Use temporary directories for file cache tests

### Performance Considerations

- **Avoid Map Copying**: Use pointer semantics for large structs
- **Lock Granularity**: Per-key locking for cache operations
- **Memory Optimization**: Use sync.Pool for frequently allocated objects
- **Atomic Operations**: Prefer atomic ops over mutex for simple metrics

## Frontend Development

### SvelteKit Setup

- **Framework**: SvelteKit with TypeScript
- **Build Tool**: Vite with custom plugins
- **Adapter**: Static adapter for embedding in Go binary
- **Styling**: Dark-first theme with CSS custom properties

### Color Scheme (Dark Mode)

```css
:root {
  --primary-color: #40798c;
  --secondary-color: #70a9a1;
  --background-color: #1a1a1a;
  --text-color: #e9ecef;
  --border-color: #495057;
}
```

### Build Integration

- Frontend builds to `webserver/dashboard/frontend/build/`
- Embedded using `//go:embed` directive
- Served as static files via `http.FileServer`

## API Endpoints

### Metrics API

Example endpoints for metrics:

- `GET /metrics` - All metrics
- `GET /metrics/cache` - Cache-specific metrics
- `GET /metrics/timing` - Timing metrics

  **Response Format**: JSON with atomic value serialization

### Dashboard Routes

- `/` - Main dashboard interface
- Static assets served from embedded filesystem

## Development Commands

### Go Development

```bash
go run main.go                          # Run proxy
go run main.go -listen=:8080            # Custom port
go build -o apt_cacher_go.exe           # Build executable
```

### Frontend Development

```bash
cd webserver/dashboard/frontend
pnpm install                            # Install dependencies
pnpm run dev                            # Development server
pnpm run build                          # Production build
```

## Configuration

### Command Line Arguments

- `--listen` (default: ":9999") - Proxy listen address
- `--ca-cert` (default: "ssl/ca.crt") - CA certificate path
- `--ca-key` (default: "ssl/ca.key") - CA private key path
- `--cache-dir` (default: "var/cache/") - Cache directory
- `--webserver-listen` (default: "localhost:8080") - Dashboard address

### Configuration File

- Location: `var/config.json`
- Format: JSON with version management
- Custom types: Uses ByteSize and Duration parsing

## Security Considerations

### Certificate Authority

- Generate CA certificate for HTTPS interception
- Clients must trust the CA certificate
- Store CA files securely (ssl/ directory)

### MITM Proxy Operation

- Intercepts HTTPS traffic for caching
- Generates certificates on-demand
- Validates cache directives and headers

## Performance Tuning

### Cache Configuration

- `MaxCacheSize`: Set appropriate cache size limit
- `CacheCleanupInterval`: Adjust cleanup frequency
- `DefaultCacheMaxAge`: Default expiration time

### Eviction Strategy

- Hybrid LRU + size-based scoring
- Evicts to 80% of max size to avoid thrashing
- Considers both access time and file size

## Troubleshooting

### Common Issues

1. **Certificate Errors**: Ensure CA certificate is trusted by clients
2. **Cache Corruption**: Delete `var/` directory and restart
3. **Memory Usage**: Monitor cache size and adjust limits
4. **Concurrent Access**: Check for race conditions with `-race` flag

### Debug Logging

- All operations include structured logging
- Cache operations log with key and size information
- Metrics updates are logged for verification

## Extension Points

### Adding New Metrics

1. Add field to appropriate metrics struct
2. Use atomic operations for updates
3. Add JSON marshaling tag
4. Update API endpoints if needed

### Custom Cache Policies

1. Implement cache directive parsing
2. Add configuration options
3. Update shouldCache logic
4. Consider backward compatibility

### New API Endpoints

1. Implement endpoint interface
2. Add to API registration
3. Follow JSON response patterns
4. Add appropriate error handling

Remember to maintain thread safety, proper error handling, and consistent logging throughout all modifications.
