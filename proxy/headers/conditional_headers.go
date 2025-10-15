package headers

import (
	"net/http"
	"reservoir/utils/typeutils"
	"time"
)

type eTag = string

type conditionalHeaders struct {
	IfModifiedSince   typeutils.Optional[time.Time]
	IfUnmodifiedSince typeutils.Optional[time.Time]
	IfNoneMatch       typeutils.Optional[eTag]
	IfMatch           typeutils.Optional[eTag]
	IfRange           typeutils.Optional[typeutils.Either[eTag, time.Time]]
}

// Strips the conditionals (except If-Range) present in conditionalHeaders from the given HTTP header map.
func (ch *conditionalHeaders) StripFromHeader(header http.Header) {
	header.Del("If-Modified-Since")
	header.Del("If-Unmodified-Since")
	header.Del("If-None-Match")
	header.Del("If-Match")

	// We need to keep If-Range for Range requests
}
