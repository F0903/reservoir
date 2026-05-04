package cache

import "io"

type sizeHintReader interface {
	SizeHint() int64
}

type sizedReader struct {
	io.Reader
	size int64
}

func (r sizedReader) SizeHint() int64 {
	return r.size
}

// WithSizeHint attaches an expected byte length to a reader without changing the cache interface.
func WithSizeHint(reader io.Reader, size int64) io.Reader {
	if size < 0 {
		return reader
	}
	return sizedReader{Reader: reader, size: size}
}

func ReaderSizeHint(reader io.Reader) (int64, bool) {
	if hinted, ok := reader.(sizeHintReader); ok {
		size := hinted.SizeHint()
		return size, size >= 0
	}
	if remaining, ok := reader.(interface{ Len() int }); ok {
		return int64(remaining.Len()), true
	}
	return 0, false
}
