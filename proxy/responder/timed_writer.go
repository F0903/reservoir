package responder

import (
	"io"
	"time"
)

type timedWriter struct {
	writer   io.Writer
	duration time.Duration
}

func (w *timedWriter) Write(p []byte) (int, error) {
	start := time.Now()
	n, err := w.writer.Write(p)
	w.duration += time.Since(start)
	return n, err
}
