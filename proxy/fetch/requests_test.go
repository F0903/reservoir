package fetch

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestRefuseInternalAddress(t *testing.T) {
	tests := []struct {
		address string
		refused bool
	}{
		{"127.0.0.1:80", true},
		{"127.8.8.8:443", true},
		{"[::1]:80", true},
		{"[::ffff:127.0.0.1]:80", true},
		{"169.254.169.254:80", true},
		{"[fe80::1%eth0]:80", true},
		{"0.0.0.0:80", true},
		{"not-an-address", true},
		{"10.0.0.5:80", false},
		{"192.168.1.20:3142", false},
		{"93.184.216.34:443", false},
		{"[2606:2800:220:1::1]:443", false},
	}
	for _, tt := range tests {
		err := refuseInternalAddress("tcp", tt.address, nil)
		if refused := err != nil; refused != tt.refused {
			t.Errorf("%s: refused=%v (err=%v), want %v", tt.address, refused, err, tt.refused)
		}
	}
}

func TestChangeRequestToTargetPicksScheme(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		tls     bool
		upgrade bool
		want    string
	}{
		{"tunnel is always https", "/file", true, false, "https"},
		{"plain http stays http", "http://example.test/file", false, false, "http"},
		{"plain http is upgraded", "http://example.test/file", false, true, "https"},
		{"absolute https stays https", "https://example.test/file", false, false, "https"},
		{"no scheme means http", "/file", false, false, "http"},
		{"no scheme is upgraded", "/file", false, true, "https"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.ParseRequestURI(tt.target)
			if err != nil {
				t.Fatal(err)
			}
			req := &http.Request{URL: parsed, Host: "example.test"}
			if tt.tls {
				req.TLS = &tls.ConnectionState{}
			}

			if err := changeRequestToTarget(req, tt.upgrade); err != nil {
				t.Fatalf("changeRequestToTarget failed: %v", err)
			}
			if req.URL.Scheme != tt.want {
				t.Fatalf("scheme = %q, want %q", req.URL.Scheme, tt.want)
			}
		})
	}
}

type stalledBody struct {
	stop chan struct{}
}

func (b *stalledBody) Read([]byte) (int, error) {
	<-b.stop
	return 0, errors.New("cancelled")
}

func (b *stalledBody) Close() error { return nil }

func TestIdleTimeoutBodyCancelsStalledRead(t *testing.T) {
	stalled := &stalledBody{stop: make(chan struct{})}
	cancelled := make(chan struct{})
	body := newIdleTimeoutBody(stalled, 20*time.Millisecond, func() {
		close(stalled.stop)
		close(cancelled)
	})

	go body.Read(make([]byte, 1))

	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected a stalled read to cancel the request")
	}
}

type instantBody struct{}

func (instantBody) Read(p []byte) (int, error) { return len(p), nil }
func (instantBody) Close() error               { return nil }

func TestIdleTimeoutBodyIgnoresSlowConsumer(t *testing.T) {
	cancels := 0
	body := newIdleTimeoutBody(instantBody{}, 20*time.Millisecond, func() { cancels++ })

	if _, err := body.Read(make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond) // Time spent writing to the client must not count as idle.
	if cancels != 0 {
		t.Fatal("request was cancelled between reads")
	}

	body.Close()
	if cancels != 1 {
		t.Fatalf("expected Close to release the request context, got %d cancels", cancels)
	}
}

var _ io.ReadCloser = (*idleTimeoutBody)(nil)
