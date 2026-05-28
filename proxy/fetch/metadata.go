package fetch

import (
	"net/http"
	"time"
)

type CacheMetadata struct {
	ETag         string
	LastModified time.Time
	Header       http.Header
	Vary         []string
}
