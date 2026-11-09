package httplistener

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestRunForceClosesWhenShutdownTimesOut(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := probe.Addr().String()
	probe.Close()

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	listener := New(address, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		started <- struct{}{}
		<-release
	}))
	listener.shutdownTimeout = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- listener.Run(ctx) }()

	go func() {
		for {
			if resp, err := http.Get("http://" + address); err == nil {
				resp.Body.Close()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Millisecond):
			}
		}
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("request never reached the handler")
	}
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("expected a clean stop after force close, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after the shutdown timeout")
	}
}
