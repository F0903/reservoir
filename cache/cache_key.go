package cache

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/crypto/blake2b"
)

type CacheKey struct {
	Hex string
}

func NewCacheKey(bytes []byte) CacheKey {
	hashBytes := blake2b.Sum256(bytes)
	return CacheKey{
		Hex: hex.EncodeToString(hashBytes[:]),
	}
}

func FromString(input string) CacheKey {
	return NewCacheKey([]byte(input))
}

func MakeFromRequest(r *http.Request) CacheKey {
	scheme := "http"
	if r.URL.Scheme == "http" || r.URL.Scheme == "https" {
		scheme = r.URL.Scheme
	} else if r.TLS != nil {
		scheme = "https"
	}
	normHost := strings.ToLower(r.Host)
	// The path is used exactly as sent: /dir and /dir/, or %2F and /, can be different resources.
	stringKey := fmt.Sprintf("%s|%s|%s|%s|%s", scheme, r.Method, normHost, r.URL.EscapedPath(), r.URL.RawQuery)
	slog.Debug("Creating cache key", "key", stringKey)
	return FromString(stringKey)
}

func (ck *CacheKey) String() string {
	return ck.Hex
}
