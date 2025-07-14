package counting_reader

import (
	"io"
)

type CountingReader struct {
	reader io.Reader
	read   *int
}

func New(reader io.Reader, read *int) CountingReader {
	return CountingReader{
		reader: reader,
		read:   read,
	}
}

func (cw CountingReader) ResetCount() {
	*cw.read = 0
}

func (cw CountingReader) GetCount() int {
	return *cw.read
}

func (cw CountingReader) Read(p []byte) (n int, err error) {
	n, err = cw.reader.Read(p)
	*cw.read += n
	return n, err
}
