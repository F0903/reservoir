package log

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"reservoir/config"
	"reservoir/webserver/api/apitypes"
	"time"
)

type LogStreamEndpoint struct{}

func (m *LogStreamEndpoint) Path() string {
	return "/log/stream"
}

func (m *LogStreamEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *LogStreamEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	//TODO: refactor and seperate generic SSE logic

	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")

	slog.Debug("starting SSE stream")

	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("response writer does not support flushing, so can't use SSE", "content-type", header.Get("Content-Type"))
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	cfg := config.Get()
	logFilePath := cfg.LogFile.Read()
	if logFilePath == "" {
		http.Error(w, "no log file configured", http.StatusNotFound)
		return
	}

	openStat := func() (*os.File, int64, error) {
		f, err := os.Open(logFilePath)
		if err != nil {
			return nil, 0, err
		}

		st, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, 0, err
		}

		return f, st.Size(), nil
	}

	f, offset, err := openStat()
	if err != nil {
		http.Error(w, "failed to open log file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	var partial []byte // carry incomplete line between reads

	writeLine := func(line []byte) error {
		// SSE frame: data:<line>\n\n

		// Avoid CR in Windows \r\n
		line = bytes.TrimRight(line, "\r\n")
		if len(line) == 0 {
			return nil
		}

		if _, err := w.Write([]byte("data: ")); err != nil {
			return err
		}
		if _, err := w.Write(line); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n\n")); err != nil {
			return err
		}

		flusher.Flush()
		return nil
	}

	for {
		select {
		case <-r.Context().Done():
			slog.Debug("SSE stream done")
			return

		case <-heartbeat.C:
			// SSE heartbeat (comment format)
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
			slog.Debug("sent SSE stream heartbeat")

		case <-ticker.C:
			slog.Debug("running SSE tick")

			logStat, err := os.Stat(logFilePath)
			if err != nil {
				// transient error; try again on next tick
				slog.Error("failed to stat log file in SSE stream", "error", err)
				continue
			}

			logSize := logStat.Size()

			// Rotation/truncate: size shrank -> reopen, reset
			if logSize < offset {
				slog.Debug("log file rotated or truncated", "log_file", logFilePath)

				f.Close()
				f, offset, err = openStat()
				if err != nil {
					continue
				}
				partial = nil
			}

			if logSize == offset {
				slog.Debug("no new data in log file", "log_file", logFilePath)
				continue // no new data
			}

			// Cap burst reads to 1MB per tick to avoid huge allocations
			writeNum := min(logSize-offset, 1<<20)

			buf := make([]byte, writeNum)
			readCount, err := f.ReadAt(buf, offset)
			if err != nil && !errors.Is(err, io.EOF) {
				slog.Error("failed to read log file in SSE stream", "error", err)
				continue
			}

			offset += int64(readCount)
			chunk := append(partial, buf[:readCount]...)

			// Split by newline; keep tail as partial if not terminated
			lines := bytes.Split(chunk, []byte{'\n'})
			if len(lines) == 0 {
				slog.Debug("no complete lines read, skipping", "log_file", logFilePath)
				continue
			}

			partial = nil
			// If last segment isn’t newline-terminated, keep it as partial
			last := lines[len(lines)-1]
			if readCount == 0 || (len(last) > 0 && chunk[len(chunk)-1] != '\n') {
				partial = last
				lines = lines[:len(lines)-1]
			}

			for _, line := range lines {
				if err := writeLine(line); err != nil {
					return
				}
			}
		}
	}
}
