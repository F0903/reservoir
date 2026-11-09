package responder

import (
	"bytes"
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
