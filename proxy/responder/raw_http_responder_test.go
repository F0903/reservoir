package responder

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func readRawResponse(t *testing.T, reader *bufio.Reader) *http.Response {
	t.Helper()

	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("failed to read raw response: %v", err)
	}
	t.Cleanup(func() {
		resp.Body.Close()
	})

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		t.Fatalf("failed to drain response body: %v", err)
	}

	return resp
}

func TestRawHTTPResponderDoesNotLeakHeadersBetweenResponses(t *testing.T) {
	var buf bytes.Buffer
	responder := NewRawHTTPResponder(&buf)

	responder.SetHeader("X-First", "1")
	if written, err := responder.Write(http.StatusOK, strings.NewReader("first")); err != nil {
		t.Fatalf("failed to write first response: %v", err)
	} else if written != int64(len("first")) {
		t.Fatalf("first response reported %d bytes written", written)
	}

	responder.SetHeader("X-Second", "2")
	if written, err := responder.Write(http.StatusOK, strings.NewReader("second")); err != nil {
		t.Fatalf("failed to write second response: %v", err)
	} else if written != int64(len("second")) {
		t.Fatalf("second response reported %d bytes written", written)
	}

	reader := bufio.NewReader(&buf)
	first := readRawResponse(t, reader)
	second := readRawResponse(t, reader)

	if got := first.Header.Get("X-First"); got != "1" {
		t.Fatalf("first response missing own header: got %q", got)
	}
	if got := first.Header.Get("X-Second"); got != "" {
		t.Fatalf("first response leaked future header: got %q", got)
	}
	if got := second.Header.Get("X-Second"); got != "2" {
		t.Fatalf("second response missing own header: got %q", got)
	}
	if got := second.Header.Get("X-First"); got != "" {
		t.Fatalf("second response leaked previous header: got %q", got)
	}
}
