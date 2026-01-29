package tests

import (
	"io"
	"net/http"
	"testing"
)

func TestHttpsMITM(t *testing.T) {
	env := SetupHttpsTestEnv(t)

	targetURL := env.Upstream.URL + "/https-test"
	resp, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("Failed to make HTTPS request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "https response body" {
		t.Errorf("Expected response body 'https response body', got '%s'", string(body))
	}
}
