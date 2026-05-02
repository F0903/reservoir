package proxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reservoir/metrics"
	proxyResponder "reservoir/proxy/responder"
	"strings"
	"testing"
	"time"
)

type delayedResponder struct {
	headers http.Header
	delay   time.Duration
}

func newDelayedResponder(delay time.Duration) *delayedResponder {
	return &delayedResponder{
		headers: make(http.Header),
		delay:   delay,
	}
}

func (r *delayedResponder) SetHeader(name string, value string) {
	r.headers.Set(name, value)
}

func (r *delayedResponder) AddHeader(name string, value string) {
	r.headers.Add(name, value)
}

func (r *delayedResponder) SetHeaders(headers http.Header) {
	r.headers = headers.Clone()
}

func (r *delayedResponder) GetHeaders() http.Header {
	return r.headers
}

func (r *delayedResponder) Write(status int, body io.Reader) (int64, time.Duration, error) {
	start := time.Now()
	if r.delay > 0 {
		time.Sleep(r.delay)
	}

	written, err := io.Copy(io.Discard, body)
	return written, time.Since(start), err
}

func (r *delayedResponder) WriteEmpty(status int) error {
	_, _, err := r.Write(status, http.NoBody)
	return err
}

func (r *delayedResponder) WriteError(message string, errorCode int) error {
	return nil
}

func (r *delayedResponder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, proxyResponder.ErrHijackNotSupported
}

func useFreshMetrics(t *testing.T) {
	t.Helper()

	previous := metrics.Global
	metrics.Global = metrics.NewMetrics()
	t.Cleanup(func() {
		metrics.Global = previous
	})
}

func TestFinalizeAndRespondTracksClientResponseLatency(t *testing.T) {
	useFreshMetrics(t)

	req := httptest.NewRequest(http.MethodGet, "http://example.test/package.deb", nil)
	responder := newDelayedResponder(2 * time.Millisecond)

	if err := finalizeAndRespond(responder, strings.NewReader("payload"), http.StatusOK, req); err != nil {
		t.Fatalf("finalizeAndRespond returned error: %v", err)
	}

	if got := metrics.Global.Requests.ClientResponseLatency.Get(); got <= 0 {
		t.Fatalf("expected client response latency to be recorded, got %d", got)
	}
	if got := metrics.Global.Requests.ClientResponses.Get(); got != 1 {
		t.Fatalf("expected 1 client response, got %d", got)
	}
	if got := metrics.Global.Requests.BytesServed.Get(); got != int64(len("payload")) {
		t.Fatalf("expected %d bytes served, got %d", len("payload"), got)
	}
}
