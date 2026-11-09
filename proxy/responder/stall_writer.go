package responder

import (
	"io"
	"time"
)

// Small enough that a client accepting data at any sensible speed finishes a piece
// well inside the timeout, however large the file is.
const stallWriteChunkSize = 64 * 1024

// StallWriter fails a write when the client accepts nothing for its timeout, so a client
// that stops reading can't hold its connection and upstream download open forever.
// The deadline is renewed for each piece, so slow but moving clients are unaffected.
type StallWriter struct {
	writer      io.Writer
	setDeadline func(time.Time) error
	timeout     time.Duration
}

func NewStallWriter(writer io.Writer, setDeadline func(time.Time) error, timeout time.Duration) *StallWriter {
	return &StallWriter{writer: writer, setDeadline: setDeadline, timeout: timeout}
}

// Rearm starts a fresh deadline.
func (w *StallWriter) Rearm() {
	_ = w.setDeadline(time.Now().Add(w.timeout)) // Not every writer supports deadlines.
}

func (w *StallWriter) Write(p []byte) (written int, err error) {
	for len(p) > 0 {
		w.Rearm()

		n, err := w.writer.Write(p[:min(len(p), stallWriteChunkSize)])
		written += n
		if err != nil {
			return written, err
		}
		p = p[n:]
	}
	return written, nil
}
