package cache

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"path"
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
	if r.TLS != nil {
		scheme = "https"
	}
	normHost := strings.ToLower(r.Host)
	normPath := path.Clean(r.URL.Path)
	stringKey := fmt.Sprintf("%s|%s|%s|%s|%s", scheme, r.Method, normHost, normPath, r.URL.RawQuery)
	log.Printf("Creating cache key: %s", stringKey)
	return FromString(stringKey)
}

func (ck *CacheKey) String() string {
	return ck.Hex
}

func (ck *CacheKey) Bytes() ([]byte, error) {
	hashBytes, err := hex.DecodeString(ck.Hex)
	if err != nil {
		return nil, fmt.Errorf("error decoding cache key: %v", err)
	}
	return hashBytes, nil
}
