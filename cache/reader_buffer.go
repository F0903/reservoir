package cache

import (
	"bytes"
	"errors"
	"io"
)

const initialCacheReadBufferSize = 1024

func ReadAllCacheBytes(data io.Reader, limit int64) ([]byte, error) {
	initialCapacity := initialCacheReadBufferSize
	maxInt := int64(^uint(0) >> 1)
	if sizeHint, ok := ReaderSizeHint(data); ok && sizeHint >= 0 && sizeHint <= limit && sizeHint <= maxInt {
		return readAllExactSizeHint(data, int(sizeHint))
	}

	buf := bytes.NewBuffer(make([]byte, 0, initialCapacity))
	_, err := buf.ReadFrom(data)
	return buf.Bytes(), err
}

func readAllExactSizeHint(data io.Reader, size int) ([]byte, error) {
	buf := make([]byte, size)
	if size > 0 {
		if _, err := io.ReadFull(data, buf); err != nil {
			return nil, err
		}
	}

	var extra [1]byte
	n, err := io.ReadFull(data, extra[:])
	if errors.Is(err, io.EOF) {
		return buf, nil
	}
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}
	if n == 0 {
		return buf, nil
	}

	out := bytes.NewBuffer(make([]byte, 0, size+n+initialCacheReadBufferSize))
	out.Write(buf)
	out.Write(extra[:n])
	_, err = out.ReadFrom(data)
	return out.Bytes(), err
}
