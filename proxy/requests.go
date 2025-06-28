package proxy

import (
	"fmt"
	"log"
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

func addrToUrl(addr string) (*url.URL, error) {
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "https://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func changeRequestToTarget(req *http.Request) error {
	targetHost := req.Host
	targetUrl, err := addrToUrl(targetHost)
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

func sendRequestToTarget(req *http.Request) (*http.Response, error) {
	// Change request URL to point to the target server.
	changeRequestToTarget(req)
	// Remove hop-by-hop headers in the request that should not be forwarded to the target server.
	removeHopByHopHeaders(req.Header)

	log.Printf("Sending request %v", req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to target (%v): %w", req.URL, err)
	}
	resp.Body.Close()
	log.Printf("Sent request to target %v, got response status: %s", req.URL, resp.Status)

	// Remove any hop-by-hop headers in the response that should not be forwarded to the client.
	removeHopByHopHeaders(resp.Header)

	return resp, nil
}
