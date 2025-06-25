package proxy

import (
	"fmt"
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

func changeRequestToTarget(req *http.Request, targetHost string) error {
	targetUrl, err := addrToUrl(targetHost)
	if err != nil {
		return fmt.Errorf("invalid target host '%s': %v", targetHost, err)
	}

	targetUrl.Path = req.URL.Path
	targetUrl.RawQuery = req.URL.RawQuery
	req.URL = targetUrl
	// Make sure this is unset for sending the request through a client
	req.RequestURI = ""

	return nil
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
