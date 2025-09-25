package log

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"reservoir/config"
	"reservoir/utils"
	"reservoir/webserver/api/apitypes"
	"reservoir/webserver/streaming"
	"time"
)

type LogStreamEndpoint struct{}

type logStreamStore struct {
	partial       []byte // carry incomplete line between reads
	currentOffset int64  // current read offset in the log file
	logPath       string
	logFile       *os.File
}

func (m *LogStreamEndpoint) Path() string {
	return "/log/stream"
}

func (m *LogStreamEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       "GET",
			Func:         m.Get,
			RequiresAuth: true,
		},
	}
}

func (m *LogStreamEndpoint) Tick(w http.ResponseWriter, writeStream func([]byte) error, store *logStreamStore) error {
	logStat, err := os.Stat(store.logPath)
	if err != nil {
		// transient error; try again on next tick
		slog.Error("failed to stat log file in SSE stream", "error", err)
		return err
	}

	logSize := logStat.Size()

	// Rotation/truncate: size shrank -> reopen, reset
	if logSize < store.currentOffset {
		slog.Debug("log file rotated or truncated", "log_file", store.logPath)

		store.logFile.Close() // Close old one before opening new.
		store.logFile, store.currentOffset, err = utils.OpenWithSize(store.logPath)
		if err != nil {
			return err
		}
		defer store.logFile.Close()

		store.partial = nil
	}

	if logSize == store.currentOffset {
		slog.Debug("no new data in log file", "log_file", store.logPath)
		return nil
	}

	// Cap burst reads to 1MB per tick to avoid huge allocations
	writeNum := min(logSize-store.currentOffset, 1<<20)

	buf := make([]byte, writeNum)
	readCount, err := store.logFile.ReadAt(buf, store.currentOffset)
	if err != nil && !errors.Is(err, io.EOF) {
		slog.Error("failed to read log file in SSE stream", "error", err)
		return err
	}

	store.currentOffset += int64(readCount)
	chunk := append(store.partial, buf[:readCount]...)

	// Split by newline; keep tail as partial if not terminated
	lines := bytes.Split(chunk, []byte{'\n'})
	if len(lines) == 0 {
		slog.Debug("no complete lines read, skipping", "log_file", store.logPath)
		return nil
	}

	store.partial = nil
	// If last segment isn’t newline-terminated, keep it as partial
	last := lines[len(lines)-1]
	if readCount == 0 || (len(last) > 0 && chunk[len(chunk)-1] != '\n') {
		store.partial = last
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		if err := writeStream(line); err != nil {
			return err
		}
	}

	return nil
}

func (m *LogStreamEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	header := w.Header()

	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("response writer does not support flushing, so can't use SSE", "content-type", header.Get("Content-Type"))
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	cfgLock := config.Global.Immutable()

	var logFilePath string
	cfgLock.Read(func(c *config.Config) {
		logFilePath = c.LogFile.Read()
	})

	if logFilePath == "" {
		http.Error(w, "no log file configured", http.StatusNotFound)
		return
	}

	f, offset, err := utils.OpenWithSize(logFilePath)
	if err != nil {
		http.Error(w, "failed to open log file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	store := logStreamStore{
		partial:       nil,
		currentOffset: offset,
		logPath:       logFilePath,
		logFile:       f,
	}
	sse := streaming.NewSseStream(header, w, flusher, 1*time.Second, 500*time.Millisecond, r.Context(), m, store)
	defer sse.Close()
	if err := sse.Start(); err != nil {
		slog.Error("SSE stream failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
