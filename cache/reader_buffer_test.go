package cache

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type trackingReader struct {
	reader io.Reader
	reads  int
}

func (r *trackingReader) Read(p []byte) (int, error) {
	r.reads++
	return r.reader.Read(p)
}

func TestReadAllCacheBytesRejectsOversizedHintWithoutReading(t *testing.T) {
	reader := &trackingReader{reader: bytes.NewReader([]byte("oversized"))}

	_, err := ReadAllCacheBytes(WithSizeHint(reader, 9), 4)
	if !errors.Is(err, ErrCacheMemoryExceeded) {
		t.Fatalf("expected ErrCacheMemoryExceeded, got %v", err)
	}
	if reader.reads != 0 {
		t.Fatalf("expected oversized size hint to fail without consuming body, got %d reads", reader.reads)
	}
}
