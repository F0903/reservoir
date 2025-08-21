package log

import (
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"reservoir/logging"
	"reservoir/webserver/api/apitypes"
)

type LogsEndpoint struct{}

func (m *LogsEndpoint) Path() string {
	return "/log"
}

func (m *LogsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *LogsEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	logFile, err := logging.OpenLogFile(true)
	if err != nil {
		if errors.Is(err, logging.ErrNoLogFile) {
			slog.Warn("tried to call /log but no log file is configured")
			http.Error(w, "no log file configured", http.StatusNotFound)
			return
		}
		slog.Error("failed to open log file", "error", err)
		http.Error(w, "failed to open log file", http.StatusInternalServerError)
		return
	}
	defer logFile.Close()

	logFileStat, err := logFile.Stat()
	if err != nil {
		http.Error(w, "failed to stat log file", http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(logFile.Name())

	http.ServeContent(w, r, filename, logFileStat.ModTime(), logFile)
}

//TODO: Add SSE streaming endpoint for "tailing" the log
