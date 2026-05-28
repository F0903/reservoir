package fetch

import (
	"io"
	"net/http"
	"reservoir/cache"
	"time"
)

type Type int

const (
	TypeCached Type = iota
	TypeDirect
)

type Status int

const (
	StatusMiss Status = iota
	StatusRevalidated
	StatusHit
	StatusStale
)

type Info struct {
	UpstreamStatus  int // Only valid if Status is StatusMiss or StatusRevalidated
	Status          Status
	UpstreamLatency time.Duration
}

// Represents a fetch that was not served from cache, but returned directly from the origin server.
type DirectResult struct {
	Info
	Response *http.Response
}

// Represents a fetch that was served from cache. Possibly revalidated from origin.
type CachedResult struct {
	Info
	Entry     *cache.Entry[CacheMetadata]
	Coalesced bool
}

type Result struct {
	Type   Type
	Cached CachedResult // Only valid if Type is TypeCached
	Direct DirectResult // Only valid if Type is TypeDirect
}

func newDirectResult(resp *http.Response, status Status, upstreamLatency time.Duration) Result {
	return Result{
		Type: TypeDirect,
		Direct: DirectResult{
			Info: Info{
				UpstreamStatus:  resp.StatusCode,
				Status:          status,
				UpstreamLatency: upstreamLatency,
			},
			Response: resp,
		},
	}
}

func newCachedResult(entry *cache.Entry[CacheMetadata], status Status, upstreamStatus int, upstreamLatency time.Duration) Result {
	return Result{
		Type: TypeCached,
		Cached: CachedResult{
			Info: Info{
				UpstreamStatus:  upstreamStatus,
				Status:          status,
				UpstreamLatency: upstreamLatency,
			},
			Entry: entry,
		},
	}
}

// FetchInfo returns the cache/origin status metadata for the selected result.
func (f *Result) FetchInfo() Info {
	switch f.Type {
	case TypeDirect:
		return f.Direct.Info
	case TypeCached:
		return f.Cached.Info
	}
	return Info{}
}

func (f *Result) Response() (data io.ReadCloser, header http.Header, status int) {
	switch f.Type {
	case TypeDirect:
		return f.Direct.Response.Body, f.Direct.Response.Header, f.Direct.UpstreamStatus
	case TypeCached:
		status := f.Cached.UpstreamStatus
		if status == 0 {
			status = http.StatusOK
		}
		return f.Cached.Entry.Data, f.Cached.Entry.Metadata.Object.Header, status
	}
	return nil, nil, 0
}
