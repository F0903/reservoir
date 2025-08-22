package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reservoir/config"
	"reservoir/utils/assertedpath"
)

var (
	ErrNoLogFile = errors.New("no log file configured")
)

var logLevel slog.LevelVar

func OpenLogFile(readonly bool) (*os.File, error) {
	config := config.Get()

	logFilePath := config.LogFile.Read()
	if logFilePath == "" {
		return nil, ErrNoLogFile
	}

	assertedPath, err := assertedpath.TryAssert(logFilePath)
	if err != nil {
		return nil, err
	}

	var perms os.FileMode
	var flags int

	if readonly {
		flags = os.O_RDONLY
		perms = 0444
	} else {
		flags = os.O_RDWR | os.O_APPEND | os.O_CREATE
		perms = 0644
	}

	logFile, err := os.OpenFile(assertedPath.Path, flags, perms)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return logFile, nil
}

func initLogFileWriter(writers *[]io.Writer) {
	slog.Info("Initializing log file writer...")

	logFile, err := OpenLogFile(false)
	if err != nil {
		if errors.Is(err, ErrNoLogFile) {
			slog.Info("No log file configured, skipping log file writer initialization")
			return
		}
		panic(fmt.Errorf("failed to open log file: %v", err))
	}
	// Don't close File since we are handing it to slog

	*writers = append(*writers, logFile)
	slog.Info("Log file writer added successfully", "path", logFile.Name())
}

func SetLogLevel(level slog.Level) {
	logLevel.Set(level)
}

func Init() {
	config := config.Get()

	slog.Info("Initializing logging...")

	slog.Info("Setting up log writers...")
	slog.Info("Added Stdout writer")
	var writers []io.Writer = []io.Writer{
		os.Stdout,
	}
	initLogFileWriter(&writers)

	level := config.LogLevel.Read()
	slog.Info("Setting log level", "log-level", level)
	logLevel.Set(level)

	slog.Info("Setting up slog handler...")
	mw := io.MultiWriter(writers...)
	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: &logLevel,
	})
	slog.SetDefault(slog.New(handler))
	slog.Info("Slog handler and multi-writer set up successfully", "log-level", level)
}
