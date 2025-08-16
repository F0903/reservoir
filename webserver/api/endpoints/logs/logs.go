package logs

import (
	"net/http"
	"path/filepath"
	"reservoir/logging"
	"reservoir/webserver/api/apitypes"
)

type LogsEndpoint struct{}

func (m *LogsEndpoint) Path() string {
	return "/logs"
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
	logFile := logging.OpenLogFile()
	defer logFile.Close()

	logFileStat, err := logFile.Stat()
	if err != nil {
		http.Error(w, "failed to stat log file", http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(logFile.Name())

	http.ServeContent(w, r, filename, logFileStat.ModTime(), logFile)
}
