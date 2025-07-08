package proxy

import (
	"apt_cacher_go/config"
	"apt_cacher_go/utils/optional"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type cacheControl struct {
	noCache bool
	maxAge  time.Duration
}

type conditionalHeaders struct {
	ifModifiedSince   optional.Optional[time.Time]
	ifUnmodifiedSince optional.Optional[time.Time]
	ifNoneMatch       optional.Optional[string]
	ifMatch           optional.Optional[string]
	ifRange           optional.Optional[string] // Currently not used, but can be extended for range requests
}

func (ch *conditionalHeaders) removeFromHeader(header http.Header) {
	if ch.ifModifiedSince.IsSome() {
		header.Del("If-Modified-Since")
	}
	if ch.ifUnmodifiedSince.IsSome() {
		header.Del("If-Unmodified-Since")
	}
	if ch.ifNoneMatch.IsSome() {
		header.Del("If-None-Match")
	}
	if ch.ifMatch.IsSome() {
		header.Del("If-Match")
	}
	if ch.ifRange.IsSome() {
		header.Del("If-Range")
	}
}

type cacheDirective struct {
	conditionalHeaders conditionalHeaders
	cacheControl       optional.Optional[cacheControl]
	rangeHeader        optional.Optional[string] // Currently not used, but can be extended for range requests
	expires            optional.Optional[time.Time]
}

func (cd *cacheDirective) shouldCache() bool {
	if cd.cacheControl.IsSome() {
		cc := cd.cacheControl.ForceUnwrap()
		if cc.noCache {
			return false // No caching allowed
		}

		if cc.maxAge < 1 {
			return false // If max-age is less than 1 second, treat it as no-cache
		}
	}

	if cd.expires.IsSome() {
		expires := cd.expires.ForceUnwrap()
		if expires.Before(time.Now()) {
			return false // If the Expires header is in the past, do not cache
		}
	}

	return true // If no cache control or expires headers prevent caching, we can cache
}

func (cd *cacheDirective) getExpiresOrDefault() time.Time {
	if !config.Global.ForceDefaultCacheMaxAge {
		if cd.cacheControl.IsSome() {
			cc := cd.cacheControl.ForceUnwrap()
			if cc.maxAge > 0 {
				return time.Now().Add(cc.maxAge)
			}
		}
		if cd.expires.IsSome() {
			return *cd.expires.ForceUnwrap()
		}
	}

	defaultMaxAge := config.Global.DefaultCacheMaxAge.Cast()
	return time.Now().Add(defaultMaxAge)
}

func parseCacheControl(ccHeader string) (*cacheControl, error) {
	cc := &cacheControl{}
	// Parse the Cache-Control header for max-age directive
	for directive := range strings.SplitSeq(ccHeader, ",") {
		directive = strings.TrimSpace(directive)
		if directive == "no-cache" || directive == "no-store" {
			cc.noCache = true
		} else if after, ok := strings.CutPrefix(directive, "max-age="); ok {
			// max-age directive specifies the maximum amount of time a response is considered fresh in seconds.
			maxAge, err := strconv.ParseInt(after, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse max-age: %v", err)
			}
			if maxAge < 1 {
				cc.noCache = true // If max-age is less than 1, treat it as no-cache
				log.Printf("max-age is less than 1 second, treating as no-cache")
				continue
			}
			cc.maxAge = time.Duration(maxAge) * time.Second
		}
	}

	return cc, nil
}

func parseCacheDirective(header http.Header) *cacheDirective {
	cd := &cacheDirective{}
	for key, values := range header {
		value := values[0] // Header.Get also uses the first value, so we do the same here
		switch key {
		case "If-Modified-Since":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				cd.conditionalHeaders.ifModifiedSince = optional.Some(&t)
			}
		case "If-Unmodified-Since":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				cd.conditionalHeaders.ifUnmodifiedSince = optional.Some(&t)
			}
		case "If-None-Match":
			if value != "" {
				cd.conditionalHeaders.ifNoneMatch = optional.Some(&value)
			}
		case "If-Match":
			if value != "" {
				cd.conditionalHeaders.ifMatch = optional.Some(&value)
			}
		case "If-Range":
			if value != "" {
				cd.conditionalHeaders.ifRange = optional.Some(&value)
			}
		case "Range":
			if value != "" {
				cd.rangeHeader = optional.Some(&value)
			}
		case "Cache-Control":
			if cc, err := parseCacheControl(value); err == nil {
				cd.cacheControl = optional.Some(cc)
			}
		case "Expires":
			if t, err := time.Parse(http.TimeFormat, value); err == nil {
				cd.expires = optional.Some(&t)
			}
		}
	}
	return cd
}

func removeUnsupportedHeaders(header http.Header) {
	unsupportedHeaders := []string{
		"Range",         // Range requests are currently not supported in this proxy
		"If-Range",      // Range requests are currently not supported in this proxy
		"Accept-Ranges", // Range requests are currently not supported in this proxy
	}
	for _, h := range unsupportedHeaders {
		header.Del(h)
	}
}
