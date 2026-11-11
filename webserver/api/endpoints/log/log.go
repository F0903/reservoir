package log

import (
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	appservices "reservoir/application/services"
	"reservoir/webserver/api/apihttp"
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
	logFile, err := ctx.Services.Config.OpenLogFileRead()
	if err != nil {
		if errors.Is(err, appservices.ErrNoLogFile) {
			slog.Warn("tried to call /log but no log file is configured")
			apihttp.Error(w, "No Log File Configured", http.StatusNotFound)
			return
		}
		slog.Error("failed to open log file", "error", err)
		apihttp.InternalServerError(w)
		return
	}
	defer logFile.Close()

	logFileStat, err := logFile.Stat()
	if err != nil {
		slog.Error("failed to stat log file", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	filename := filepath.Base(logFile.Name())

	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, filename, logFileStat.ModTime(), logFile)
}
