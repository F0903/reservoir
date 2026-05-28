package fetch

import (
	"net/http"
	"reservoir/cache"
	"strings"
)

type quietMetadataCache interface {
	GetMetadataQuiet(key cache.CacheKey) (meta *cache.EntryMetadata[CacheMetadata], stale bool, err error)
}

func normalizeVaryHeaderValues(values []string) string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		parts := make([]string, 0)
		for part := range strings.SplitSeq(value, ",") {
			part = strings.ToLower(strings.TrimSpace(part))
			if part == "" {
				continue
			}
			parts = append(parts, part)
		}
		normalized = append(normalized, strings.Join(parts, ","))
	}
	return strings.Join(normalized, ",")
}

func makeVariantCacheKey(req *http.Request, baseKey cache.CacheKey, vary []string) cache.CacheKey {
	if len(vary) == 0 {
		return baseKey
	}

	parts := []string{baseKey.Hex}
	for _, headerName := range vary {
		values := req.Header.Values(http.CanonicalHeaderKey(headerName))
		parts = append(parts, headerName+"="+normalizeVaryHeaderValues(values))
	}
	return cache.FromString(strings.Join(parts, "|"))
}

func (f *Fetcher) lookupCacheKey(req *http.Request, baseKey cache.CacheKey) cache.CacheKey {
	if vary, ok := f.variantIndex.Get(baseKey); ok {
		return makeVariantCacheKey(req, baseKey, vary)
	}

	candidate := makeVariantCacheKey(req, baseKey, supportedVaryHeaders)
	if candidate != baseKey && f.cacheEntryExists(candidate) {
		f.setVariantIndex(baseKey, supportedVaryHeaders)
		return candidate
	}

	return baseKey
}

func (f *Fetcher) singleflightKey(req *http.Request, baseKey cache.CacheKey) string {
	return makeVariantCacheKey(req, baseKey, supportedVaryHeaders).Hex
}

func (f *Fetcher) setVariantIndex(baseKey cache.CacheKey, vary []string) {
	if len(vary) == 0 {
		f.variantIndex.Delete(baseKey)
		return
	}

	copied := make([]string, len(vary))
	copy(copied, vary)
	f.variantIndex.Set(baseKey, copied)
}

func (f *Fetcher) cacheEntryExists(key cache.CacheKey) bool {
	quietCache, ok := f.cache.(quietMetadataCache)
	if !ok {
		return false
	}

	_, _, err := quietCache.GetMetadataQuiet(key)
	return err == nil
}
