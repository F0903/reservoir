package log

import (
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"reservoir/logging"
	"reservoir/webserver/api/apitypes"
)

type LogEndpoint struct{}

func (m *LogEndpoint) Path() string {
	return "/log"
}

func (m *LogEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       "GET",
			Func:         m.Get,
			RequiresAuth: true,
		},
	}
}

func (m *LogEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	logFile, err := logging.OpenLogFileRead()
	if err != nil {
		if errors.Is(err, logging.ErrNoLogFile) {
			slog.Warn("tried to call /log but no log file is configured")
			http.Error(w, "No Log File Configured", http.StatusNotFound)
			return
		}
		slog.Error("failed to open log file", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer logFile.Close()

	logFileStat, err := logFile.Stat()
	if err != nil {
		slog.Error("failed to stat log file", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(logFile.Name())

	r.Header.Set("Cache-Control", "no-store")
	http.ServeContent(w, r, filename, logFileStat.ModTime(), logFile)
}
