package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/proxy/responder"
	"reservoir/utils/typeutils"
	"strconv"
	"strings"
	"time"
)

type hitStatus int

const (
	hitStatusMiss hitStatus = iota
	hitStatusRevalidated
	hitStatusHit
)

type fwdReason int

const (
	fwdReasonMiss fwdReason = iota
	fwdReasonBypass
	fwdReasonStale
)

type cacheStatus struct {
	hitStatus hitStatus
	fwdReason typeutils.Optional[fwdReason]
	fwdStatus typeutils.Optional[int]
	stored    bool
}

func makeCacheStatusHeader(cached typeutils.Optional[cache.Entry[cachedRequestInfo]], cacheStatus cacheStatus) string {
	slog.Debug("Making Cache-Status header...", "cached", cached, "cacheStatus", cacheStatus)

	params := make([]string, 0, 6)
	params = append(params, "reservoir")

	switch cacheStatus.hitStatus {
	case hitStatusHit:
		params = append(params, "hit")
	case hitStatusRevalidated:
		params = append(params, "hit; detail=\"revalidated\"")
	case hitStatusMiss:
		params = append(params, "miss")
	}

	if cacheStatus.fwdReason.IsSome() {
		fwdReason := cacheStatus.fwdReason.ForceUnwrap()
		switch fwdReason {
		case fwdReasonMiss:
			params = append(params, "fwd=miss")
		case fwdReasonBypass:
			params = append(params, "fwd=bypass")
		case fwdReasonStale:
			params = append(params, "fwd=stale")
		}
	}

	if cacheStatus.fwdStatus.IsSome() {
		fwdStatus := cacheStatus.fwdStatus.ForceUnwrap()
		params = append(params, fmt.Sprintf("fwd-status=%d", fwdStatus))
	}

	if cacheStatus.stored {
		params = append(params, "stored")
	}

	if cached.IsSome() && (cacheStatus.hitStatus == hitStatusHit || cacheStatus.hitStatus == hitStatusRevalidated) {
		ttl := max(0, int(time.Until(cached.ForceUnwrap().Metadata.Expires).Seconds()))
		params = append(params, fmt.Sprintf("ttl=%d", ttl))
	}

	headerStr := strings.Join(params, "; ")
	slog.Debug("Constructed Cache-Status header:", "header", headerStr)

	return headerStr
}

func getCurrentAge(originalHead http.Header, storedAt time.Time) int {
	slog.Debug("Calculating current age of cached response...")

	dateHeader := originalHead.Get("Date")
	upstreamAge := originalHead.Get("Age")
	now := time.Now()

	apparentAge := 0
	if parsedDateHeader, err := time.Parse(http.TimeFormat, dateHeader); err == nil {
		apparentAge = max(0, int(storedAt.Sub(parsedDateHeader).Seconds()))
	}

	correctedAge := apparentAge
	if parsedUpstreamAge, err := strconv.Atoi(upstreamAge); err == nil {
		correctedAge = max(apparentAge, parsedUpstreamAge)
	}

	responseDelay := 0 // We don't track request time

	correctedInitialAge := correctedAge + responseDelay

	residentTime := int(now.Sub(storedAt).Seconds())

	currentAge := max(0, correctedInitialAge+residentTime)
	slog.Debug("Current age of cached response:", "age", currentAge)

	return currentAge
}

func addCacheHeaders(r responder.Responder, req *http.Request, cached typeutils.Optional[cache.Entry[cachedRequestInfo]], cacheStatus cacheStatus) {
	cacheStatusHeader := makeCacheStatusHeader(cached, cacheStatus)
	r.AddHeader("Cache-Status", cacheStatusHeader)
	slog.Debug("Cache-Status header set:", "cache_status", cacheStatusHeader)

	r.AddHeader("Via", fmt.Sprintf("%s reservoir", req.Proto))

	if cached.IsSome() {
		cached := cached.ForceUnwrap()

		wasCached := cacheStatus.hitStatus == hitStatusHit || cacheStatus.hitStatus == hitStatusRevalidated
		justCached := cacheStatus.hitStatus == hitStatusHit && cacheStatus.stored
		if wasCached || justCached {
			currentAge := getCurrentAge(cached.Metadata.Object.Header, cached.Metadata.TimeWritten)
			ageStr := strconv.Itoa(currentAge)
			r.SetHeader("Age", ageStr)
		}
	}
}

func fetchResultToCacheStatus(fetched fetchResult) cacheStatus {
	isRevalidated := fetched.Cached.fetchInfo.Status == hitStatusRevalidated
	isMiss := fetched.Cached.fetchInfo.Status == hitStatusMiss

	fwdReason := typeutils.None[fwdReason]()
	if isRevalidated {
		fwdReasonNum := fwdReasonStale
		fwdReason = typeutils.Some(&fwdReasonNum)
	}

	fwdStatus := typeutils.None[int]()
	if isRevalidated || isMiss {
		fwdStatusNum := fetched.Cached.fetchInfo.UpstreamStatus
		fwdStatus = typeutils.Some(&fwdStatusNum)
	}

	cacheStatus := cacheStatus{
		hitStatus: fetched.Cached.Status,
		fwdReason: fwdReason,
		fwdStatus: fwdStatus,
		stored:    !fetched.Cached.Coalesced && (isMiss || isRevalidated),
	}

	return cacheStatus
}
