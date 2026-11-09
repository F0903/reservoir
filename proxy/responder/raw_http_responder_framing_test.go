package responder

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func assertNoBodyFraming(t *testing.T, raw string) {
	t.Helper()
	if strings.Contains(raw, "Transfer-Encoding") {
		t.Fatalf("unexpected chunked framing: %q", raw)
	}
	if !strings.HasSuffix(raw, "\r\n\r\n") || strings.HasSuffix(raw, "0\r\n\r\n") {
		t.Fatalf("response must end right after its headers: %q", raw)
	}
}

func TestRawHTTPResponderBodylessStatusesHaveNoFraming(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, status := range []int{http.StatusNoContent, http.StatusNotModified} {
			var buf bytes.Buffer
			responder := NewRawHTTPResponder(&buf)
			responder.SetRequest(&http.Request{Method: method})

			if _, _, err := responder.Write(status, http.NoBody); err != nil {
				t.Fatalf("%s %d: write failed: %v", method, status, err)
			}

			raw := buf.String()
			assertNoBodyFraming(t, raw)
			if strings.Contains(raw, "Content-Length") {
				t.Fatalf("%s %d: must not send Content-Length: %q", method, status, raw)
			}
		}
	}
}

func TestRawHTTPResponderHeadWithUnknownSizeHasNoChunkTerminator(t *testing.T) {
	var buf bytes.Buffer
	responder := NewRawHTTPResponder(&buf)
	responder.SetRequest(&http.Request{Method: http.MethodHead})

	if _, _, err := responder.Write(http.StatusOK, http.NoBody); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	raw := buf.String()
	if strings.HasSuffix(raw, "0\r\n\r\n") {
		t.Fatalf("HEAD response must not carry a chunk terminator: %q", raw)
	}
	if strings.Contains(raw, "Connection: close") {
		t.Fatalf("HEAD response must keep the connection open: %q", raw)
	}
}
