package responder

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestStallWriterRearmsDeadlineForEachPiece(t *testing.T) {
	var buf bytes.Buffer
	deadlines := 0
	writer := NewStallWriter(&buf, func(time.Time) error {
		deadlines++
		return nil
	}, time.Minute)

	data := make([]byte, 3*stallWriteChunkSize+1)
	n, err := writer.Write(data)
	if err != nil || n != len(data) {
		t.Fatalf("wrote %d bytes, err=%v", n, err)
	}
	if buf.Len() != len(data) {
		t.Fatalf("underlying writer got %d bytes", buf.Len())
	}
	if deadlines != 4 {
		t.Fatalf("expected a fresh deadline for each of 4 pieces, got %d", deadlines)
	}
}

type failingWriter struct{ accepted int }

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.accepted > 0 {
		return 0, errors.New("write failed")
	}
	w.accepted = len(p)
	return len(p), nil
}

func TestStallWriterStopsAtFirstError(t *testing.T) {
	underlying := &failingWriter{}
	writer := NewStallWriter(underlying, func(time.Time) error { return nil }, time.Minute)

	n, err := writer.Write(make([]byte, 3*stallWriteChunkSize))
	if err == nil {
		t.Fatal("expected the write error to be returned")
	}
	if n != stallWriteChunkSize {
		t.Fatalf("expected the bytes accepted before the failure, got %d", n)
	}
}
