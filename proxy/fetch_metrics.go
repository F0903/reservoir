package proxy

import (
	"io"
	"reservoir/metrics"
)

type fetchedBytesReadCloser struct {
	io.ReadCloser
	metrics *metrics.Metrics
}

func (r fetchedBytesReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.metrics.Requests.BytesFetched.Add(int64(n))
	}
	return n, err
}

func trackFetchedBytes(body io.ReadCloser, m *metrics.Metrics) io.ReadCloser {
	return fetchedBytesReadCloser{ReadCloser: body, metrics: metrics.Ensure(m)}
}
