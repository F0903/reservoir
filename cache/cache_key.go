package cache

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"path"
	"strings"

	"golang.org/x/crypto/blake2b"
)

type CacheKey struct {
	hashBytes []byte
	hashHex   string
}

func NewCacheKey(bytes []byte) *CacheKey {
	hashBytes := blake2b.Sum256(bytes)
	return &CacheKey{
		hashBytes: hashBytes[:],
		hashHex:   hex.EncodeToString(hashBytes[:]),
	}
}

func FromString(input string) *CacheKey {
	return NewCacheKey([]byte(input))
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

func (ck *CacheKey) Bytes() []byte {
	return ck.hashBytes
}

func (ck *CacheKey) Hex() string {
	return ck.hashHex
}

func (ck *CacheKey) String() string {
	return ck.hashHex
}
