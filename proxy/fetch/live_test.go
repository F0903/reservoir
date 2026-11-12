package fetch

import (
	"io"
	"reservoir/cache"
	"strings"
	"testing"
	"time"
)

const liveTestBody = "0123456789abcdef"

func newTestLiveBody(window int) *liveBody {
	return newLiveBody(cache.FromString("live-test"), cache.EntryMetadata[CacheMetadata]{Size: int64(len(liveTestBody))}, window)
}

// resumeFromTestBody stands in for the cache or upstream and records how it was asked.
func resumeFromTestBody(gotOff *int64, gotCached *bool) resumeFunc {
	return func(off int64, cached bool) (io.ReadCloser, error) {
		*gotOff, *gotCached = off, cached
		return io.NopCloser(strings.NewReader(liveTestBody[off:])), nil
	}
}

func TestLiveReaderFollowsDownload(t *testing.T) {
	body := newTestLiveBody(64)
	reader := body.newReader(nil)

	body.Write([]byte(liveTestBody[:6]))
	first := make([]byte, 6)
	if _, err := io.ReadFull(reader, first); err != nil || string(first) != liveTestBody[:6] {
		t.Fatalf("got %q (err=%v) before the download finished", first, err)
	}

	go func() {
		body.Write([]byte(liveTestBody[6:]))
		body.settle(nil, true)
		body.finish()
	}()
	rest, err := io.ReadAll(reader)
	if err != nil || string(rest) != liveTestBody[6:] {
		t.Fatalf("got %q (err=%v) for the rest", rest, err)
	}
}

func TestLiveReaderLeftBehindContinuesFromCache(t *testing.T) {
	body := newTestLiveBody(8)
	var off int64
	var cached bool
	reader := body.newReader(resumeFromTestBody(&off, &cached))

	body.Write([]byte(liveTestBody[:4]))
	first := make([]byte, 4)
	if _, err := io.ReadFull(reader, first); err != nil {
		t.Fatal(err)
	}
	// The download runs ahead and the window moves past what the reader still needs.
	for i := 4; i < len(liveTestBody); i += 4 {
		body.Write([]byte(liveTestBody[i : i+4]))
	}

	if body.newReader(nil) != nil {
		t.Fatal("expected no new reader once the start of the body has left the window")
	}

	done := make(chan string, 1)
	go func() {
		rest, _ := io.ReadAll(reader)
		done <- string(rest)
	}()
	select {
	case <-done:
		t.Fatal("the reader did not wait for the download to finish")
	case <-time.After(50 * time.Millisecond):
	}

	body.settle(nil, true)
	body.finish()
	if rest := <-done; rest != liveTestBody[4:] || off != 4 || !cached {
		t.Fatalf("got %q continued at offset %d (cached=%v), want the rest from the cache at offset 4", rest, off, cached)
	}
}

func TestLiveReaderContinuesElsewhereWhenDownloadFails(t *testing.T) {
	body := newTestLiveBody(64)
	var off int64
	cached := true
	reader := body.newReader(resumeFromTestBody(&off, &cached))

	body.Write([]byte(liveTestBody[:4]))
	body.finish() // Never settled: the download broke off.

	all, err := io.ReadAll(reader)
	if err != nil || string(all) != liveTestBody || off != 4 || cached {
		t.Fatalf("got %q (err=%v) continued at offset %d (cached=%v), want the whole body continued at offset 4", all, err, off, cached)
	}
}

func TestLiveReaderGetsLastByteOnlyOnceDownloadIsFinished(t *testing.T) {
	body := newTestLiveBody(64)
	reader := body.newReader(nil)
	body.Write([]byte(liveTestBody))
	body.settle(nil, true)

	allButLast := make([]byte, len(liveTestBody)-1)
	if _, err := io.ReadFull(reader, allButLast); err != nil || string(allButLast) != liveTestBody[:len(liveTestBody)-1] {
		t.Fatalf("got %q (err=%v) before the download finished", allButLast, err)
	}

	done := make(chan string, 1)
	go func() {
		last, _ := io.ReadAll(reader)
		done <- string(last)
	}()
	select {
	case <-done:
		t.Fatal("the reader got the last byte before the download was finished")
	case <-time.After(50 * time.Millisecond):
	}

	body.finish()
	if last := <-done; last != liveTestBody[len(liveTestBody)-1:] {
		t.Fatalf("got %q for the last byte", last)
	}
}
