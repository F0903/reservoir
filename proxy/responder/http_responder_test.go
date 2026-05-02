package responder

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type delayedReader struct {
	reader *bytes.Reader
	delay  time.Duration
	slept  bool
}

func (r *delayedReader) Read(p []byte) (int, error) {
	if !r.slept {
		time.Sleep(r.delay)
		r.slept = true
	}

	return r.reader.Read(p)
}

func TestHTTPResponderWriteDurationExcludesBodyReadTime(t *testing.T) {
	const body = "response body"
	readDelay := 50 * time.Millisecond

	rec := httptest.NewRecorder()
	responder := NewHTTPResponder(rec)
	bodyReader := &delayedReader{
		reader: bytes.NewReader([]byte(body)),
		delay:  readDelay,
	}

	start := time.Now()
	written, writeDuration, err := responder.Write(http.StatusOK, bodyReader)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("failed to write response: %v", err)
	}

	if written != int64(len(body)) {
		t.Fatalf("expected %d bytes written, got %d", len(body), written)
	}
	if elapsed < readDelay {
		t.Fatalf("expected elapsed time to include read delay %v, got %v", readDelay, elapsed)
	}
	if writeDuration >= readDelay/2 {
		t.Fatalf("expected write duration to exclude read delay %v, got %v", readDelay, writeDuration)
	}
}
