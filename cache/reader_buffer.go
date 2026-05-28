package cache

import (
	"bytes"
	"io"
)

const (
	initialCacheReadBufferSize = 1024
	maxCacheReadProbeLimit     = int64(^uint64(0) >> 1)
)

func ReadAllCacheBytes(data io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, ErrCacheMemoryExceeded
	}

	initialCapacity, err := cacheReadBufferCapacity(data, limit)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(make([]byte, 0, initialCapacity))
	limited := &io.LimitedReader{R: data, N: cacheReadProbeLimit(limit)}
	_, err = buf.ReadFrom(limited)
	if err != nil {
		return nil, err
	}
	if int64(buf.Len()) > limit {
		return nil, ErrCacheMemoryExceeded
	}
	return buf.Bytes(), nil
}

func cacheReadBufferCapacity(data io.Reader, limit int64) (int, error) {
	sizeHint, ok := ReaderSizeHint(data)
	if !ok || sizeHint < 0 {
		return initialCacheReadBufferSize, nil
	}
	if sizeHint > limit {
		return 0, ErrCacheMemoryExceeded
	}

	maxInt := int64(^uint(0) >> 1)
	if sizeHint > maxInt {
		return initialCacheReadBufferSize, nil
	}
	return int(sizeHint), nil
}

func cacheReadProbeLimit(limit int64) int64 {
	if limit == maxCacheReadProbeLimit {
		return limit
	}
	return limit + 1
}
