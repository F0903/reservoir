package headers

import (
	"errors"
	"net/http"
	"reservoir/config"
	"reservoir/utils/typeutils"
	"time"
)

var (
	ErrParseMaxAge = errors.New("failed to parse max-age")
)

type HeaderDirectives struct {
	ConditionalHeaders conditionalHeaders
	CacheControl       typeutils.Optional[cacheControl]
	Range              typeutils.Optional[rangeHeader]
	Expires            typeutils.Optional[time.Time]
}

func parseIfRange(value string) typeutils.Optional[typeutils.Either[eTag, time.Time]] {
	if value == "" {
		return typeutils.None[typeutils.Either[eTag, time.Time]]()
	}

	if t, err := time.Parse(http.TimeFormat, value); err == nil {
		timeIfRange := typeutils.Right[eTag, time.Time](&t)
		return typeutils.Some(&timeIfRange)
	}

	etagIfRange := typeutils.Left[eTag, time.Time](&value)
	return typeutils.Some(&etagIfRange)
}

func ParseHeaderDirective(header http.Header) *HeaderDirectives {
	hd := &HeaderDirectives{}
	for key, values := range header {
		value := values[0] // Header.Get also uses the first value, so we do the same here
		switch key {
		case "If-Modified-Since":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				hd.ConditionalHeaders.IfModifiedSince = typeutils.Some(&t)
			}
		case "If-Unmodified-Since":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				hd.ConditionalHeaders.IfUnmodifiedSince = typeutils.Some(&t)
			}
		case "If-None-Match":
			if value != "" {
				hd.ConditionalHeaders.IfNoneMatch = typeutils.Some(&value)
			}
		case "If-Match":
			if value != "" {
				hd.ConditionalHeaders.IfMatch = typeutils.Some(&value)
			}
		case "If-Range":
			if value != "" {
				hd.ConditionalHeaders.IfRange = parseIfRange(value)
			}
		case "Range":
			if rh, err := parseRangeHeader(value); err == nil {
				hd.Range = typeutils.Some(&rh)
			}
		case "Cache-Control":
			if cc, err := parseCacheControl(value); err == nil {
				hd.CacheControl = typeutils.Some(cc)
			}
		case "Expires":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				hd.Expires = typeutils.Some(&t)
			}
		}
	}
	return hd
}

func (hd *HeaderDirectives) ShouldCache() bool {
	cfgLock := config.Global.Immutable()
	var ignoreCacheControl bool
	cfgLock.Read(func(c *config.Config) {
		ignoreCacheControl = c.IgnoreCacheControl.Read()
	})

	if !ignoreCacheControl && hd.CacheControl.IsSome() {
		cc := hd.CacheControl.ForceUnwrap()
		if cc.noCache {
			return false // No caching allowed
		}

		if cc.maxAge < 1 {
			return false // If max-age is less than 1 second, treat it as no-cache
		}
	}

	if hd.Expires.IsSome() {
		expires := hd.Expires.ForceUnwrap()
		if expires.Before(time.Now()) {
			return false // If the Expires header is in the past, do not cache
		}
	}

	if hd.Range.IsSome() {
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
		if hd.CacheControl.IsSome() {
			cc := hd.CacheControl.ForceUnwrap()
			if cc.maxAge > 0 {
				return time.Now().Add(cc.maxAge)
			}
		}
		if hd.Expires.IsSome() {
			return hd.Expires.ForceUnwrap()
		}
	}

	return time.Now().Add(defaultCacheMaxAge)
}
