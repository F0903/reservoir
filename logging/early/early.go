// Import this package in packages that have global variables that log in their initializer.
// This allows the EarlyBuffer to capture logs before the actual logging setup is complete.
// And then write them afterwards to the actual final log handler.

package early

import (
	"bytes"
	"log/slog"
)

var EarlyBuffer *bytes.Buffer = setupIntermediateBuffer()

func setupIntermediateBuffer() *bytes.Buffer {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("Set up intermediate buffer to catch early logs before switching to the configured log handler.")

	return &buf
}
