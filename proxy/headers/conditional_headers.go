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

// Strips the conditionals present in conditionalHeaders from the given HTTP header map.
func (ch *conditionalHeaders) StripFromHeader(header http.Header) {
	if ch.IfModifiedSince.IsSome() {
		header.Del("If-Modified-Since")
	}
	if ch.IfUnmodifiedSince.IsSome() {
		header.Del("If-Unmodified-Since")
	}
	if ch.IfNoneMatch.IsSome() {
		header.Del("If-None-Match")
	}
	if ch.IfMatch.IsSome() {
		header.Del("If-Match")
	}
	if ch.IfRange.IsSome() {
		header.Del("If-Range")
	}
}
