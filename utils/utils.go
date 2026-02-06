package utils

import (
	"errors"
	"log/slog"
	"os"
	"strconv"
)

var ErrNotImplemented = errors.New("not implemented")

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

// Creates an index from the first 8 hexadecimal characters of a string.
func Hex8ToIndex(s string) uint32 {
	// We use the first 8 hex chars (4 bytes / 32 bits) to calculate the index.
	// This supports any value up to 2^32.
	var val uint32
	for i := 0; i < 8 && i < len(s); i++ {
		h := s[i]
		var b uint32
		switch {
		case h >= '0' && h <= '9':
			b = uint32(h - '0')
		case h >= 'a' && h <= 'f':
			b = uint32(h - 'a' + 10)
		case h >= 'A' && h <= 'F':
			b = uint32(h - 'A' + 10)
		}
		val = (val << 4) | b
	}
	return val
}
