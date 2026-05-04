package proxy

import (
	"net/http"
	"reservoir/cache"
	"strings"
)

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

func (f *fetcher) lookupCacheKey(req *http.Request, baseKey cache.CacheKey) cache.CacheKey {
	vary, _ := f.variantIndex.Get(baseKey)
	return makeVariantCacheKey(req, baseKey, vary)
}

func (f *fetcher) singleflightKey(req *http.Request, baseKey cache.CacheKey) string {
	return makeVariantCacheKey(req, baseKey, supportedVaryHeaders).Hex
}

func (f *fetcher) setVariantIndex(baseKey cache.CacheKey, vary []string) {
	if len(vary) == 0 {
		f.variantIndex.Delete(baseKey)
		return
	}

	copied := make([]string, len(vary))
	copy(copied, vary)
	f.variantIndex.Set(baseKey, copied)
}
