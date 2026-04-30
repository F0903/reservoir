package proxy

import (
	"net/http"
	"reservoir/config"
	"reservoir/proxy/headers"
	"slices"
	"strings"
	"time"
)

var supportedVaryHeaders = []string{"accept-encoding"}

type cachePolicy struct {
	cfg *config.Config
}

type cacheDecision struct {
	Cacheable bool
	Expires   time.Time
	Reason    string
	Vary      []string
}

func newCachePolicy(cfg *config.Config) cachePolicy {
	return cachePolicy{cfg: cfg}
}

func parseVaryHeaders(header http.Header) []string {
	vary := make([]string, 0)
	for _, rawHeader := range header.Values("Vary") {
		for rawValue := range strings.SplitSeq(rawHeader, ",") {
			name := strings.ToLower(strings.TrimSpace(rawValue))
			if name == "" {
				continue
			}
			vary = append(vary, name)
		}
	}
	slices.Sort(vary)
	vary = slices.Compact(vary)
	return vary
}

func supportsVary(vary []string) bool {
	for _, name := range vary {
		if name == "*" {
			return false
		}
		if !slices.Contains(supportedVaryHeaders, name) {
			return false
		}
	}
	return true
}

func varyContains(vary []string, headerName string) bool {
	return slices.Contains(vary, strings.ToLower(headerName))
}

func (p cachePolicy) RequestAllowsSharedCache(req *http.Request) bool {
	return req.Header.Get("Authorization") == "" && req.Header.Get("Cookie") == ""
}

func (p cachePolicy) Decide(req *http.Request, resp *http.Response, upstreamHd *headers.HeaderDirectives) cacheDecision {
	if req.Method != http.MethodGet {
		return cacheDecision{Cacheable: false, Reason: "request method is not GET"}
	}

	if resp.StatusCode != http.StatusOK {
		return cacheDecision{Cacheable: false, Reason: "response status is not 200 OK"}
	}

	if !p.RequestAllowsSharedCache(req) {
		return cacheDecision{Cacheable: false, Reason: "request contains credentials"}
	}

	if len(resp.Header.Values("Set-Cookie")) > 0 {
		return cacheDecision{Cacheable: false, Reason: "response sets cookies"}
	}

	vary := parseVaryHeaders(resp.Header)
	if !supportsVary(vary) {
		return cacheDecision{Cacheable: false, Reason: "response uses unsupported Vary"}
	}
	if resp.Header.Get("Content-Encoding") != "" && !varyContains(vary, "accept-encoding") {
		return cacheDecision{Cacheable: false, Reason: "encoded response does not vary by Accept-Encoding"}
	}

	ignoreCacheControl := p.cfg.Proxy.CachePolicy.IgnoreCacheControl.Read()
	if !upstreamHd.ShouldCache(ignoreCacheControl) {
		return cacheDecision{Cacheable: false, Reason: "response cache directives disallow storage"}
	}

	expires := upstreamHd.GetExpiresOrDefault(
		p.cfg.Proxy.CachePolicy.ForceDefaultMaxAge.Read(),
		p.cfg.Proxy.CachePolicy.DefaultMaxAge.Read().Cast(),
	)

	return cacheDecision{
		Cacheable: true,
		Expires:   expires,
		Reason:    "cacheable",
		Vary:      vary,
	}
}
