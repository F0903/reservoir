package utils

import (
	"log/slog"
	"os"
	"strconv"
)

func OpenWithSize(path string) (*os.File, int64, error) {
	f, err := os.Open(path)
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

func StringToLogLevel(logLevel string) slog.Level {
	var level slog.Level
	quoted := strconv.Quote(logLevel)
	level.UnmarshalJSON([]byte(quoted))
	return level
}
