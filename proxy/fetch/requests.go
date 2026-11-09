package fetch

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

var (
	ErrInvalidTargetHost  = errors.New("invalid target host")
	ErrSendRequestFailed  = errors.New("error sending request to target")
	ErrBlockedDestination = errors.New("destination address is not allowed")
)

const (
	upstreamDialTimeout           = 30 * time.Second
	upstreamKeepAlive             = 30 * time.Second
	upstreamTLSHandshakeTimeout   = 10 * time.Second
	upstreamResponseHeaderTimeout = 30 * time.Second
	upstreamBodyIdleTimeout       = 60 * time.Second
	upstreamIdleConnTimeout       = 90 * time.Second
	upstreamExpectContinueTimeout = 1 * time.Second
)

// refuseInternalAddress stops proxy clients reaching this host or cloud metadata services.
// It checks the resolved IP at connect time, so a DNS name pointing at loopback is refused too.
func refuseInternalAddress(_, address string, _ syscall.RawConn) error {
	addrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBlockedDestination, err)
	}

	ip := addrPort.Addr().Unmap()
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("%w: %s", ErrBlockedDestination, ip)
	}
	return nil
}

func newUpstreamClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   upstreamDialTimeout,
				KeepAlive: upstreamKeepAlive,
				Control:   refuseInternalAddress,
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

// idleTimeoutBody cancels the upstream request when one read stalls, so a silent
// upstream can't hold a download and everyone waiting on it forever. Only time
// inside Read counts, so a slow client doesn't trip it.
type idleTimeoutBody struct {
	io.ReadCloser
	timeout time.Duration
	timer   *time.Timer
	cancel  func()
}

func newIdleTimeoutBody(body io.ReadCloser, timeout time.Duration, cancel func()) *idleTimeoutBody {
	timer := time.AfterFunc(timeout, cancel)
	timer.Stop()
	return &idleTimeoutBody{ReadCloser: body, timeout: timeout, timer: timer, cancel: cancel}
}

func (b *idleTimeoutBody) Read(p []byte) (int, error) {
	b.timer.Reset(b.timeout)
	defer b.timer.Stop()
	return b.ReadCloser.Read(p)
}

func (b *idleTimeoutBody) Close() error {
	b.timer.Stop()
	err := b.ReadCloser.Close()
	b.cancel()
	return err
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

func upstreamScheme(req *http.Request, upgradeHTTP bool) string {
	if req.TLS != nil || upgradeHTTP {
		return "https"
	}
	if req.URL.Scheme != "" {
		return req.URL.Scheme
	}
	return "http"
}

func changeRequestToTarget(req *http.Request, upgradeHTTP bool) error {
	targetUrl, err := url.Parse(upstreamScheme(req, upgradeHTTP) + "://" + req.Host)
	if err != nil {
		slog.Error("Invalid target host", "host", req.Host, "error", err)
		return fmt.Errorf("%w: %v", ErrInvalidTargetHost, err)
	}

	targetUrl.Path = req.URL.Path
	targetUrl.RawPath = req.URL.RawPath
	targetUrl.RawQuery = req.URL.RawQuery
	targetUrl.Fragment = req.URL.Fragment
	req.URL = targetUrl
	// Make sure this is unset for sending the request through a client
	req.RequestURI = ""

	return nil
}

func sendRequestToTarget(client *http.Client, req *http.Request, upgradeHTTP bool) (*http.Response, error) {
	// Change request URL to point to the target server.
	if err := changeRequestToTarget(req, upgradeHTTP); err != nil {
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
