package hybrid

import (
	"io"
	"log/slog"
	"time"
)

const minDemotionInterval = 10 * time.Millisecond
const maxDemotionInterval = time.Minute

func demotionInterval(demoteAfter time.Duration) time.Duration {
	interval := demoteAfter / 2
	if interval < minDemotionInterval {
		return minDemotionInterval
	}
	if interval > maxDemotionInterval {
		return maxDemotionInterval
	}
	return interval
}

func (c *Cache[MetadataT]) demoteIdleEntries() {
	demoteAfter := time.Duration(c.demoteAfter.Get())
	if demoteAfter <= 0 {
		return
	}

	cutoff := time.Now().Add(-demoteAfter)
	c.demoteEntriesOlderThan(cutoff)
	c.enforceMaxCacheSize()
}

func (c *Cache[MetadataT]) demoteEntriesOlderThan(cutoff time.Time) {
	for _, key := range c.memory.DemotionCandidates(cutoff) {
		if err := c.memory.DemoteEntry(key, cutoff, func(data io.Reader, expires time.Time, metadata MetadataT) error {
			demoted, err := c.file.Cache(key, data, expires, metadata)
			if demoted != nil && demoted.Data != nil {
				_ = demoted.Data.Close()
			}
			return err
		}); err != nil {
			slog.Debug("Failed to demote memory cache entry", "key", key.Hex, "error", err)
		}
	}
}
