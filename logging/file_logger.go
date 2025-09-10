package logging

import "github.com/DeRuina/timberjack"

// Currently just a thin wrapper around timberjack to allow future customization if needed.
type fileLogger struct {
	path string
	tj   *timberjack.Logger
}

func newFileLogger(path string, maxSizeMb int, maxBackups int, compress bool) *fileLogger {
	tj := &timberjack.Logger{
		Filename:   path,
		MaxSize:    maxSizeMb,
		MaxBackups: maxBackups,
		Compress:   compress,
		LocalTime:  true,
	}
	return &fileLogger{
		path: path,
		tj:   tj,
	}
}

func (fl *fileLogger) Path() string {
	return fl.path
}

func (fl *fileLogger) Write(p []byte) (n int, err error) {
	return fl.tj.Write(p)
}

func (fl *fileLogger) Close() error {
	return fl.tj.Close()
}
