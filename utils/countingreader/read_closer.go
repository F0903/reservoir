package countingreader

import "io"

type CountingReadCloser struct {
	CountingReader
	closer io.Closer
}

func NewReadCloser(readCloser io.ReadCloser, read *int) *CountingReadCloser {
	return &CountingReadCloser{
		CountingReader: New(readCloser, read),
		closer:         readCloser,
	}
}

func (crc *CountingReadCloser) Close() error {
	return crc.closer.Close()
}
