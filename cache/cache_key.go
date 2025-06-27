package cache

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"golang.org/x/crypto/blake2b"
)

type CacheKey struct {
	hashBytes []byte
}

func FromString(input string) *CacheKey {
	sum := blake2b.Sum256([]byte(input))
	return &CacheKey{
		hashBytes: sum[:],
	}
}

func MakeFromRequest(r *http.Request) *CacheKey {
	// Use scheme, host, path, query, and method
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	normHost := strings.ToLower(r.Host)
	normPath := path.Clean(r.URL.Path)
	stringKey := fmt.Sprintf("%s|%s|%s|%s|%s", scheme, normHost, r.Method, normPath, r.URL.RawQuery)
	return FromString(stringKey)
}

func (ck *CacheKey) Hash() []byte {
	return ck.hashBytes
}

func (ck *CacheKey) HashString() string {
	return string(ck.Hash())
}
