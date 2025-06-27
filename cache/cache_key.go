package cache

import (
	"crypto"
	"fmt"
	"net/http"
	"path"
	"strings"
)

// We just manage the hash function here, it's much easier to manage and
// less error-prone compared with letting it be passed in as a parameter
var hasher = crypto.BLAKE2b_256.New()

type CacheKey struct {
	hashBytes []byte
}

func FromString(input string) *CacheKey {
	return &CacheKey{
		hashBytes: hasher.Sum([]byte(input)),
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
