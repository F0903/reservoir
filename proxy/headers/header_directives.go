package headers

import (
	"errors"
	"log/slog"
	"net/http"
	"reservoir/utils/typeutils"
	"time"
)

var (
	ErrParseMaxAge      = errors.New("failed to parse max-age")
	ErrHeaderNotPresent = errors.New("header value is not present")
)

type eTag = string

type Header[T any] struct {
	name  string
	value typeutils.Optional[T]
}

func NewHeader[T any](name string, value typeutils.Optional[T]) Header[T] {
	return Header[T]{name: name, value: value}
}

func (h *Header[T]) IsPresent() bool {
	return h.value.IsSome()
}

// Removes the matching header from the given HTTP header map and sets the value of this Header to nil.
func (h *Header[T]) SyncRemove(headers http.Header) {
	if h.value.IsNone() {
		return
	}

	delete(headers, h.name)
	h.value = typeutils.None[T]()
	slog.Debug("Removed header from request:", "header", h.name)
}

func (h *Header[T]) Value() T {
	return h.value.ForceUnwrap()
}

type HeaderDirectives struct {
	CacheControl      Header[cacheControl]
	Expires           Header[time.Time]
	Range             Header[rangeHeader]
	IfModifiedSince   Header[time.Time]
	IfUnmodifiedSince Header[time.Time]
	IfNoneMatch       Header[eTag]
	IfMatch           Header[eTag]
	IfRange           Header[typeutils.Either[eTag, time.Time]]
}

func ParseHeaderDirective(header http.Header) *HeaderDirectives {
	hd := &HeaderDirectives{
		CacheControl:      NewHeader("Cache-Control", typeutils.None[cacheControl]()),
		Expires:           NewHeader("Expires", typeutils.None[time.Time]()),
		Range:             NewHeader("Range", typeutils.None[rangeHeader]()),
		IfModifiedSince:   NewHeader("If-Modified-Since", typeutils.None[time.Time]()),
		IfUnmodifiedSince: NewHeader("If-Unmodified-Since", typeutils.None[time.Time]()),
		IfNoneMatch:       NewHeader("If-None-Match", typeutils.None[eTag]()),
		IfMatch:           NewHeader("If-Match", typeutils.None[eTag]()),
		IfRange:           NewHeader("If-Range", typeutils.None[typeutils.Either[eTag, time.Time]]()),
	}
	for key, values := range header {
		value := values[0] // Header.Get also uses the first value, so we do the same here
		switch key {
		case "If-Modified-Since":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				hd.IfModifiedSince.value = typeutils.Some(t)
			} else {
				slog.Debug("Error parsing If-Modified-Since header", "error", err, "value", value)
			}
		case "If-Unmodified-Since":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				hd.IfUnmodifiedSince.value = typeutils.Some(t)
			} else {
				slog.Debug("Error parsing If-Unmodified-Since header", "error", err, "value", value)
			}
		case "If-None-Match":
			if value != "" {
				hd.IfNoneMatch.value = typeutils.Some(value)
			}
		case "If-Match":
			if value != "" {
				hd.IfMatch.value = typeutils.Some(value)
			}
		case "If-Range":
			if value != "" {
				if t, err := time.Parse(http.TimeFormat, value); err == nil {
					timeIfRange := typeutils.Right[eTag](t)
					hd.IfRange.value = typeutils.Some(timeIfRange)
					continue
				}

				etagIfRange := typeutils.Left[eTag, time.Time](value)
				hd.IfRange.value = typeutils.Some(etagIfRange)
			}
		case "Range":
			if rh, err := parseRangeHeader(value); err == nil {
				hd.Range.value = typeutils.Some(rh)
			} else {
				slog.Debug("Error parsing Range header", "error", err, "value", value)
			}
		case "Cache-Control":
			if cc, err := parseCacheControl(value); err == nil {
				hd.CacheControl.value = typeutils.Some(cc)
			} else {
				slog.Debug("Error parsing Cache-Control header", "error", err, "value", value)
			}
		case "Expires":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				hd.Expires.value = typeutils.Some(t)
			} else {
				slog.Debug("Error parsing Expires header", "error", err, "value", value)
			}
		}
	}
	return hd
}

// Strips the conditionals (except If-Range) present in HeaderDirectives from the given HTTP header map.
func (hd *HeaderDirectives) StripRegularConditionals(header http.Header) {
	hd.IfModifiedSince.SyncRemove(header)
	hd.IfUnmodifiedSince.SyncRemove(header)
	hd.IfNoneMatch.SyncRemove(header)
	hd.IfMatch.SyncRemove(header)

	// We need to keep If-Range for Range requests
}

func (hd *HeaderDirectives) ShouldCache(ignoreCacheControl bool) bool {
	if !ignoreCacheControl && hd.CacheControl.IsPresent() {
		cc := hd.CacheControl.Value()
		if cc.noCache {
			return false // No caching allowed
		}

		if cc.maxAge < 1 {
			return false // If max-age is less than 1 second, treat it as no-cache
		}
	}

	if !ignoreCacheControl && hd.Expires.IsPresent() {
		expires := hd.Expires.Value()
		if expires.Before(time.Now()) {
			return false // If the Expires header is in the past, do not cache
		}
	}

	if hd.Range.IsPresent() {
		return false // Do not cache responses to Range requests
	}

	return true // If no cache control or expires headers prevent caching, we can cache
}

func (hd *HeaderDirectives) GetExpiresOrDefault(forceDefaultCacheMaxAge bool, defaultCacheMaxAge time.Duration) time.Time {
	if !forceDefaultCacheMaxAge {
		if hd.CacheControl.IsPresent() {
			cc := hd.CacheControl.Value()
			if cc.maxAge > 0 {
				return time.Now().Add(cc.maxAge)
			}
		}
		if hd.Expires.IsPresent() {
			return hd.Expires.Value()
		}
	}

	return time.Now().Add(defaultCacheMaxAge)
}
