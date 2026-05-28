package fetch

import "errors"

var (
	errUpdateCacheMetadata = errors.New("error updating cache metadata")
	errCacheResponseFailed = errors.New("error caching response")
)
