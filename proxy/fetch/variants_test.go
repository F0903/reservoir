package fetch

import (
	"fmt"
	"reservoir/cache"
	"reservoir/utils/syncmap"
	"testing"
)

func TestVariantIndexIsCapped(t *testing.T) {
	f := &Fetcher{variantIndex: syncmap.New[cache.CacheKey, []string]()}

	for i := range maxVariantIndexEntries + 10 {
		f.setVariantIndex(cache.FromString(fmt.Sprint("key-", i)), supportedVaryHeaders)
	}

	if got := f.variantIndex.Len(); got > maxVariantIndexEntries {
		t.Fatalf("variant index holds %d entries, want at most %d", got, maxVariantIndexEntries)
	}
}
