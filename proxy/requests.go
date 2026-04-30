package proxy

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidTargetHost = errors.New("invalid target host")
	ErrURLParseFailed    = errors.New("URL parse failed")
	ErrSendRequestFailed = errors.New("error sending request to target")
)

const (
	upstreamDialTimeout           = 30 * time.Second
	upstreamKeepAlive             = 30 * time.Second
	upstreamTLSHandshakeTimeout   = 10 * time.Second
	upstreamResponseHeaderTimeout = 30 * time.Second
	upstreamIdleConnTimeout       = 90 * time.Second
	upstreamExpectContinueTimeout = 1 * time.Second
)

func newUpstreamClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   upstreamDialTimeout,
				KeepAlive: upstreamKeepAlive,
			}).DialContext,
			DisableCompression:    true,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       upstreamIdleConnTimeout,
			TLSHandshakeTimeout:   upstreamTLSHandshakeTimeout,
			ResponseHeaderTimeout: upstreamResponseHeaderTimeout,
			ExpectContinueTimeout: upstreamExpectContinueTimeout,
		},
	}
}

func removeHopByHopHeaders(header http.Header) {
	for _, v := range header.Values("Connection") {
		for raw := range strings.SplitSeq(v, ",") {
			token := http.CanonicalHeaderKey(strings.TrimSpace(raw))
			if token == "" {
				continue
			}

			header.Del(token)
			slog.Debug("Removed header referenced in Connection header:", "header", token)
		}
	}

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
		slog.Error("Failed to parse URL", "addr", addr, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrURLParseFailed, err)
	}
	return u, nil
}

func changeRequestToTarget(req *http.Request, httpsDefault bool) error {
	targetHost := req.Host
	targetUrl, err := addrToUrl(targetHost, httpsDefault)
	if err != nil {
		slog.Error("Invalid target host", "host", targetHost, "error", err)
		return fmt.Errorf("%w: %v", ErrInvalidTargetHost, err)
	}

	targetUrl.Path = req.URL.Path
	targetUrl.RawQuery = req.URL.RawQuery
	targetUrl.Fragment = req.URL.Fragment
	req.URL = targetUrl
	// Make sure this is unset for sending the request through a client
	req.RequestURI = ""

	return nil
}

func sendRequestToTarget(client *http.Client, req *http.Request, httpsDefault bool) (*http.Response, error) {
	// Change request URL to point to the target server.
	if err := changeRequestToTarget(req, httpsDefault); err != nil {
		return nil, err
	}
	// Remove hop-by-hop headers in the request that should not be forwarded to the target server.
	removeHopByHopHeaders(req.Header)

	slog.Debug("Sending request", "url", req.URL, "method", req.Method)
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending request to target", "url", req.URL, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrSendRequestFailed, err)
	}
	slog.Debug("Sent request to target", "url", req.URL, "status", resp.Status)

	// Remove any hop-by-hop headers in the response that should not be forwarded to the client.
	removeHopByHopHeaders(resp.Header)

	return resp, nil
}
