package proxy

import (
	"io"
	"reservoir/metrics"
)

type fetchedBytesReadCloser struct {
	io.ReadCloser
}

func (r fetchedBytesReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		metrics.Global.Requests.BytesFetched.Add(int64(n))
	}
	return n, err
}

func trackFetchedBytes(body io.ReadCloser) io.ReadCloser {
	return fetchedBytesReadCloser{ReadCloser: body}
}
