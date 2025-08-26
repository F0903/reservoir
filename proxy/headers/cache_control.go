package headers

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

type cacheControl struct {
	noCache bool
	maxAge  time.Duration
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
				slog.Error("Failed to parse max-age", "raw", directive, "error", err)
				return nil, fmt.Errorf("%w: %v", ErrParseMaxAge, err)
			}
			if maxAge < 1 {
				cc.noCache = true // If max-age is less than 1, treat it as no-cache
				slog.Debug("max-age is less than 1 second, treating as no-cache", "raw", directive)
				continue
			}
			cc.maxAge = time.Duration(maxAge) * time.Second
		}
	}

	return cc, nil
}
