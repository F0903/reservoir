package headers

import (
	"errors"
	"log/slog"
	"net/http"
	"reservoir/config"
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
	value *T
}

func NewHeader[T any](name string, value *T) Header[T] {
	return Header[T]{name: name, value: value}
}

func (h *Header[T]) IsPresent() bool {
	return h.value != nil
}

// Removes the matching header from the given HTTP header map and sets the value of this Header to nil.
func (h *Header[T]) Remove(headers http.Header) {
	if h.value == nil {
		return
	}

	delete(headers, h.name)
	h.value = nil
	slog.Debug("Removed header from request:", "header", h.name)
}

func (h *Header[T]) Value() T {
	return *h.value
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
		CacheControl:      NewHeader[cacheControl]("Cache-Control", nil),
		Expires:           NewHeader[time.Time]("Expires", nil),
		Range:             NewHeader[rangeHeader]("Range", nil),
		IfModifiedSince:   NewHeader[time.Time]("If-Modified-Since", nil),
		IfUnmodifiedSince: NewHeader[time.Time]("If-Unmodified-Since", nil),
		IfNoneMatch:       NewHeader[eTag]("If-None-Match", nil),
		IfMatch:           NewHeader[eTag]("If-Match", nil),
		IfRange:           NewHeader[typeutils.Either[eTag, time.Time]]("If-Range", nil),
	}
	for key, values := range header {
		value := values[0] // Header.Get also uses the first value, so we do the same here
		switch key {
		case "If-Modified-Since":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				hd.IfModifiedSince.value = &t
			} else {
				slog.Error("Error parsing If-Modified-Since header", "error", err)
			}
		case "If-Unmodified-Since":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				hd.IfUnmodifiedSince.value = &t
			} else {
				slog.Error("Error parsing If-Unmodified-Since header", "error", err)
			}
		case "If-None-Match":
			if value != "" {
				hd.IfNoneMatch.value = &value
			}
		case "If-Match":
			if value != "" {
				hd.IfMatch.value = &value
			}
		case "If-Range":
			if value != "" {
				if t, err := time.Parse(http.TimeFormat, value); err == nil {
					timeIfRange := typeutils.Right[eTag](&t)
					hd.IfRange.value = &timeIfRange
				}

				etagIfRange := typeutils.Left[eTag, time.Time](&value)
				hd.IfRange.value = &etagIfRange
			}
		case "Range":
			if rh, err := parseRangeHeader(value); err == nil {
				hd.Range.value = &rh
			} else {
				slog.Error("Error parsing Range header", "error", err)
			}
		case "Cache-Control":
			if cc, err := parseCacheControl(value); err == nil {
				hd.CacheControl.value = cc
			} else {
				slog.Error("Error parsing Cache-Control header", "error", err)
			}
		case "Expires":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				hd.Expires.value = &t
			} else {
				slog.Error("Error parsing Expires header", "error", err)
			}
		}
	}
	return hd
}

// Strips the conditionals (except If-Range) present in HeaderDirectives from the given HTTP header map.
func (hd *HeaderDirectives) StripRegularConditionals(header http.Header) {
	hd.IfModifiedSince.Remove(header)
	hd.IfUnmodifiedSince.Remove(header)
	hd.IfNoneMatch.Remove(header)
	hd.IfMatch.Remove(header)

	// We need to keep If-Range for Range requests
}

func (hd *HeaderDirectives) ShouldCache() bool {
	cfgLock := config.Global.Immutable()
	var ignoreCacheControl bool
	cfgLock.Read(func(c *config.Config) {
		ignoreCacheControl = c.IgnoreCacheControl.Read()
	})

	if !ignoreCacheControl && hd.CacheControl.IsPresent() {
		cc := hd.CacheControl.value
		if cc.noCache {
			return false // No caching allowed
		}

		if cc.maxAge < 1 {
			return false // If max-age is less than 1 second, treat it as no-cache
		}
	}

	if !ignoreCacheControl && hd.Expires.IsPresent() {
		expires := hd.Expires.value
		if expires.Before(time.Now()) {
			return false // If the Expires header is in the past, do not cache
		}
	}

	if hd.Range.IsPresent() {
		return false // Do not cache responses to Range requests
	}

	return true // If no cache control or expires headers prevent caching, we can cache
}

func (hd *HeaderDirectives) GetExpiresOrDefault() time.Time {
	cfgLock := config.Global.Immutable()

	var forceDefaultCacheMaxAge bool
	var defaultCacheMaxAge time.Duration
	cfgLock.Read(func(c *config.Config) {
		forceDefaultCacheMaxAge = c.ForceDefaultCacheMaxAge.Read()
		defaultCacheMaxAge = c.DefaultCacheMaxAge.Read().Cast()
	})

	if !forceDefaultCacheMaxAge {
		if hd.CacheControl.IsPresent() {
			cc := hd.CacheControl.value
			if cc.maxAge > 0 {
				return time.Now().Add(cc.maxAge)
			}
		}
		if hd.Expires.IsPresent() {
			return *hd.Expires.value
		}
	}

	return time.Now().Add(defaultCacheMaxAge)
}
