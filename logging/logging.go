package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"reservoir/config"
	"reservoir/utils/assertedpath"
)

var logPath = assertedpath.Assert(config.Get().LogPath.Read())

func OpenLogFile() *os.File {
	f, err := os.OpenFile(logPath.Path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		panic(fmt.Errorf("failed to open log file: %v", err))
	}

	return f
}

func Init() {
	config := config.Get()

	logFile := OpenLogFile()
	mw := io.MultiWriter(os.Stdout, logFile)
	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: config.LogLevel.Read(),
	})
	slog.SetDefault(slog.New(handler))
}
