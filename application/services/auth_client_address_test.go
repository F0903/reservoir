package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"reservoir/config"
	coreauth "reservoir/webserver/auth"
)

func requestFromProxy(forwardedFor string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "10.0.0.5:1234"
	req.Header.Set("X-Forwarded-For", forwardedFor)
	return req
}

func TestRequestFromHTTPUsesForwardedClientFromTrustedProxy(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Webserver.TrustedProxies.Overwrite("10.0.0.0/8")
	svc := &AuthService{cfg: cfg, sessions: coreauth.NewSessionManager()}

	got, _ := svc.RequestFromHTTP(requestFromProxy("198.51.100.7"))
	if got.clientAddress != "198.51.100.7" {
		t.Fatalf("clientAddress = %q", got.clientAddress)
	}
}

func TestRequestFromHTTPIgnoresForwardedHeaderByDefault(t *testing.T) {
	svc := &AuthService{cfg: config.NewDefault(), sessions: coreauth.NewSessionManager()}

	got, _ := svc.RequestFromHTTP(requestFromProxy("198.51.100.7"))
	if got.clientAddress != "10.0.0.5" {
		t.Fatalf("clientAddress = %q, a forged header must not be believed", got.clientAddress)
	}
}
