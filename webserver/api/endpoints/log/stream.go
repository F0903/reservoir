package log

import (
	"bytes"
	"io"
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
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
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

	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()

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
			return

		case <-keepAlive.C:
			// SSE comment (not delivered to onmessage)
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()

		case <-ticker.C:
			st, err := os.Stat(logFilePath)
			if err != nil {
				// transient error; try again on next tick
				continue
			}

			// Rotation/truncate: size shrank -> reopen, reset
			if st.Size() < offset {
				f.Close()
				f, offset, err = openStat()
				if err != nil {
					continue
				}
				partial = nil
			}

			if st.Size() == offset {
				continue // no new data
			}

			n := st.Size() - offset
			if n > 1<<20 {
				// Cap burst reads to 1MB per tick to avoid huge allocations
				n = 1 << 20
			}

			buf := make([]byte, n)
			readN, err := f.ReadAt(buf, offset)
			if err != nil && err != io.EOF {
				continue
			}
			offset += int64(readN)
			chunk := append(partial, buf[:readN]...)

			// Split by newline; keep tail as partial if not terminated
			lines := bytes.Split(chunk, []byte{'\n'})
			if len(lines) == 0 {
				continue
			}
			partial = nil
			// If last segment isn’t newline-terminated, keep it as partial
			last := lines[len(lines)-1]
			if readN == 0 || (len(last) > 0 && chunk[len(chunk)-1] != '\n') {
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
