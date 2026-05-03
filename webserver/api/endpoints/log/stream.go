package log

import (
	"log/slog"
	"net/http"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
	"reservoir/webserver/streaming"
	"time"
)

type LogStreamEndpoint struct{}

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

func (m *LogStreamEndpoint) Tick(w http.ResponseWriter, writeStream func([]byte) error, tailer *logTailer) error {
	return tailer.Tick(writeStream)
}

func (m *LogStreamEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	header := w.Header()

	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("response writer does not support flushing, so can't use SSE", "content-type", header.Get("Content-Type"))
		apihttp.Error(w, "Streaming Unsupported", http.StatusInternalServerError)
		return
	}

	logFilePath := ctx.Config.Logging.File.Read()
	if logFilePath == "" {
		apihttp.Error(w, "No Log File Configured", http.StatusNotFound)
		return
	}

	tailer, err := newLogTailer(logFilePath)
	if err != nil {
		apihttp.InternalServerError(w)
		return
	}

	sse := streaming.NewSseStream(header, w, flusher, 1*time.Second, 500*time.Millisecond, r.Context(), m, tailer)
	defer sse.Close()
	if err := sse.Start(); err != nil {
		slog.Error("SSE stream failed", "error", err)
		apihttp.InternalServerError(w)
		return
	}
}
