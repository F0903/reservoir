package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestUsesHTTPS(t *testing.T) {
	tests := []struct {
		name        string
		requestURL  string
		headerName  string
		headerValue string
		want        bool
	}{
		{name: "nil request"},
		{name: "HTTP request", requestURL: "http://reservoir.test"},
		{name: "HTTPS request", requestURL: "https://reservoir.test", want: true},
		{name: "forwarded proto HTTPS", requestURL: "http://reservoir.test", headerName: "X-Forwarded-Proto", headerValue: "https", want: true},
		{name: "forwarded proto uses first token", requestURL: "http://reservoir.test", headerName: "X-Forwarded-Proto", headerValue: "http, https"},
		{name: "RFC forwarded HTTPS", requestURL: "http://reservoir.test", headerName: "Forwarded", headerValue: `for=192.0.2.1;proto="https";host=reservoir.test`, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.requestURL != "" {
				req = httptest.NewRequest(http.MethodGet, tt.requestURL, nil)
				if tt.headerName != "" {
					req.Header.Set(tt.headerName, tt.headerValue)
				}
			}

			if got := RequestUsesHTTPS(req); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
