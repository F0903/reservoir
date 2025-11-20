package proxy

import (
	"io"
	"net/http"
	"reservoir/cache"
	"time"
)

type fetchType int

const (
	fetchTypeCached fetchType = iota
	fetchTypeDirect
)

type fetchInfo struct {
	UpstreamStatus  int // Only valid if Status is hitStatusMiss or hitStatusRevalidated
	Status          hitStatus
	UpstreamLatency time.Duration
}

// Represents a fetch that was not served from cache, but returned directly from the origin server.
type directFetchResult struct {
	fetchInfo
	Response *http.Response
}

// Represents a fetch that was served from cache. Possibly revalidated from origin.
type cachedFetchResult struct {
	fetchInfo
	Entry     *cache.Entry[cachedRequestInfo]
	Coalesced bool
}

type fetchResult struct {
	Type   fetchType
	Cached cachedFetchResult // Only valid if Type is fetchTypeCached
	Direct directFetchResult // Only valid if Type is fetchTypeDirect
}

// Helper method to get fetchInfo
func (f *fetchResult) getFetchInfo() fetchInfo {
	switch f.Type {
	case fetchTypeDirect:
		return f.Direct.fetchInfo
	case fetchTypeCached:
		return f.Cached.fetchInfo
	}
	return fetchInfo{}
}

func (f *fetchResult) getFetchInfoRef() *fetchInfo {
	switch f.Type {
	case fetchTypeDirect:
		return &f.Direct.fetchInfo
	case fetchTypeCached:
		return &f.Cached.fetchInfo
	}
	return nil
}

func (f *fetchResult) getResponse() (data io.ReadCloser, header http.Header, status int) {
	switch f.Type {
	case fetchTypeDirect:
		return f.Direct.Response.Body, f.Direct.Response.Header, f.Direct.UpstreamStatus
	case fetchTypeCached:
		return f.Cached.Entry.Data, f.Cached.Entry.Metadata.Object.Header, f.Cached.UpstreamStatus
	}
	return nil, nil, 0
}
