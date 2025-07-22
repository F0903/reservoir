package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

func removeHopByHopHeaders(header http.Header) {
	hopHeaders := []string{
		"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate",
		"Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding", "Upgrade",
	}
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func addrToUrl(addr string, httpsDefault bool) (*url.URL, error) {
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		if httpsDefault {
			addr = "https://" + addr
		} else {
			addr = "http://" + addr
		}
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func changeRequestToTarget(req *http.Request, httpsDefault bool) error {
	targetHost := req.Host
	targetUrl, err := addrToUrl(targetHost, httpsDefault)
	if err != nil {
		return fmt.Errorf("invalid target host '%s': %v", targetHost, err)
	}

	targetUrl.Path = req.URL.Path
	targetUrl.RawQuery = req.URL.RawQuery
	targetUrl.Fragment = req.URL.Fragment
	req.URL = targetUrl
	// Make sure this is unset for sending the request through a client
	req.RequestURI = ""

	return nil
}

func sendRequestToTarget(req *http.Request, httpsDefault bool) (*http.Response, error) {
	// Change request URL to point to the target server.
	changeRequestToTarget(req, httpsDefault)
	// Remove hop-by-hop headers in the request that should not be forwarded to the target server.
	removeHopByHopHeaders(req.Header)

	slog.Debug("Sending request", "url", req.URL, "method", req.Method)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to target (%v): %w", req.URL, err)
	}
	slog.Debug("Sent request to target", "url", req.URL, "status", resp.Status)

	// Remove any hop-by-hop headers in the response that should not be forwarded to the client.
	removeHopByHopHeaders(resp.Header)

	return resp, nil
}
