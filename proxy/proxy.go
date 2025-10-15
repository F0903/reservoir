package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/certs"
	"reservoir/proxy/headers"
	"reservoir/proxy/responder"
	"reservoir/utils/httplistener"
	"reservoir/utils/typeutils"
	"time"
)

var (
	ErrCacheGetFailed       = errors.New("error getting cache for key")
	ErrNoCachedResponse304  = errors.New("received 304 Not Modified but no cached response found")
	ErrUpdateCacheMetadata  = errors.New("error updating cache metadata")
	ErrCacheResponseFailed  = errors.New("error caching response")
	ErrTLSCertFailed        = errors.New("error getting TLS certificate")
	ErrClientResponseFailed = errors.New("failed to write HTTP OK response to client")
	ErrReadRequestFailed    = errors.New("error reading request from client")
	ErrRangeNotSatisfiable  = errors.New("range not satisfiable")
	ErrIfRangeMismatch      = errors.New("If-Range header mismatch")
	ErrBadGateway           = errors.New("bad gateway. Error when sending request to upstream")
)

type cachedRequestInfo struct {
	ETag         string
	LastModified time.Time
	Header       http.Header
}

type Proxy struct {
	ca                  certs.CertAuthority
	cache               *cache.FileCache[cachedRequestInfo]
	fetch               fetcher
	retryOnInvalidRange bool
}

func (p *Proxy) Listen(address string, errChan chan error, ctx context.Context) {
	listener := httplistener.New(address, p)
	listener.ListenWithCancel(errChan, ctx)
}

// Creates a new MITM proxy. It should be passed the filenames
// for the certificate and private key of a certificate authority trusted by the
// client's machine.
func NewProxy(cacheDir string, ca certs.CertAuthority, ctx context.Context) (*Proxy, error) {
	cfgLock := config.Global.Immutable()

	var cacheCleanupInterval time.Duration
	cfgLock.Read(func(c *config.Config) {
		cacheCleanupInterval = c.CacheCleanupInterval.Read().Cast()
	})

	var retryOnInvalidRange bool
	cfgLock.Read(func(c *config.Config) {
		retryOnInvalidRange = c.RetryOnInvalidRange.Read()
	})

	cache := cache.NewFileCache[cachedRequestInfo](cacheDir, cacheCleanupInterval, ctx)
	return &Proxy{
		ca:                  ca,
		cache:               cache,
		fetch:               newFetcher(cache),
		retryOnInvalidRange: retryOnInvalidRange,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, proxyReq *http.Request) {
	r := responder.NewHTTPResponder(w)
	if proxyReq.Method == http.MethodConnect {
		if err := p.handleCONNECT(r, proxyReq); err != nil {
			slog.Error("Error handling CONNECT request", "error", err)
			return
		}
	} else {
		if err := p.handleHTTP(r, proxyReq); err != nil {
			slog.Error("Error handling HTTP request", "error", err)
			return
		}
	}
}

func finalizeAndRespond(r responder.Responder, resp io.Reader, status int, req *http.Request) error {
	body := resp
	if req.Method == http.MethodHead {
		body = http.NoBody
	}

	written, err := r.Write(status, body)
	if err != nil {
		slog.Error("Error writing response", "url", req.URL, "error", err)
		return err
	}

	metrics.Global.Requests.BytesServed.Add(written)
	slog.Info("Response sent", "url", req.URL, "bytes_written", written)
	return nil
}

func (p *Proxy) handleRangeRequest(r responder.Responder, req *http.Request, cached *cache.Entry[cachedRequestInfo], key cache.CacheKey, clientHd *headers.HeaderDirectives) error {
	rangeHeader := clientHd.Range.Value()
	start, end, err := rangeHeader.SliceSize(cached.Metadata.FileSize)
	if err != nil {
		slog.Error("Error slicing Range header", "url", req.URL, "key", key, "error", err, "range_header", rangeHeader, "file_size", cached.Metadata.FileSize)

		if !p.retryOnInvalidRange {
			r.SetHeader("Accept-Ranges", "bytes")
			r.SetHeader("Content-Range", fmt.Sprintf("bytes */%d", cached.Metadata.FileSize))
			r.WriteError("invalid Range header", http.StatusRequestedRangeNotSatisfiable)

			return ErrRangeNotSatisfiable
		}

		clientHd.Range.Remove(req.Header)
		fetched, err := p.fetch.fetchUpstream(req, clientHd, key)
		if err != nil {
			slog.Error("Error fetching resource without Range header", "url", req.URL, "key", key, "error", err)
			return err
		}

		data, header, status := fetched.getResponse()
		defer data.Close()

		r.SetHeaders(header)
		return finalizeAndRespond(r, data, status, req)
	}

	if clientHd.IfRange.IsPresent() {
		ifRange := clientHd.IfRange.Value()
		if ifRange.IsLeft() {
			// IfRange is ETag
			etagIfRange := ifRange.ForceUnwrapLeft()
			if etagIfRange != cached.Metadata.Object.ETag {
				slog.Info("If-Range does not match cached ETag. Sending full 200 response.", "url", req.URL, "key", key)
				return ErrIfRangeMismatch
			}
		} else {
			// IfRange is Time
			timeIfRange := ifRange.ForceUnwrapRight()
			if timeIfRange.Before(cached.Metadata.Object.LastModified) {
				slog.Info("If-Range does not match cached Last-Modified. Sending full 200 response.", "url", req.URL, "key", key)
				return ErrIfRangeMismatch
			}
		}

	}

	length := end - start + 1
	slog.Info("Serving Range request from cache", "url", req.URL, "key", key, "range_header", rangeHeader, "start", start, "end", end, "length", length)

	r.SetHeaders(cached.Metadata.Object.Header)
	r.SetHeader("Accept-Ranges", "bytes")
	r.SetHeader("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, cached.Metadata.FileSize))
	r.SetHeader("Content-Length", fmt.Sprintf("%d", length))
	r.SetHeader("ETag", cached.Metadata.Object.ETag)
	r.SetHeader("Last-Modified", cached.Metadata.Object.LastModified.Format(http.TimeFormat))

	sections := io.NewSectionReader(cached.Data, start, length)
	return finalizeAndRespond(r, sections, http.StatusPartialContent, req)
}

func (p *Proxy) processRequest(r responder.Responder, req *http.Request, key cache.CacheKey, clientHd *headers.HeaderDirectives) error {
	slog.Info("Processing HTTP request", "remote_addr", req.RemoteAddr, "method", req.Method, "url", req.URL)

	fetched, err := p.fetch.dedupFetch(req, key, clientHd)
	if err != nil {
		slog.Error("Error fetching resource", "url", req.URL, "key", key, "error", err)
		return err
	}

	switch fetched.Type {
	case fetchTypeDirect:
		defer fetched.Direct.Response.Body.Close()

		r.SetHeaders(fetched.Direct.Response.Header)
		if fetched.Direct.UpstreamStatus >= 200 && fetched.Direct.UpstreamStatus < 300 {
			r.SetHeader("Accept-Ranges", "bytes")
			addCacheHeaders(r, req, typeutils.None[cache.Entry[cachedRequestInfo]](), fetchResultToCacheStatus(fetched))
		}

		return finalizeAndRespond(r, fetched.Direct.Response.Body, fetched.Direct.UpstreamStatus, req)

	case fetchTypeCached:
		defer fetched.Cached.Entry.Data.Close()

		if clientHd.Range.IsPresent() {
			if err := p.handleRangeRequest(r, req, fetched.Cached.Entry, key, clientHd); err != nil {
				slog.Error("Error handling Range request", "url", req.URL, "key", key, "error", err)
				if errors.Is(err, ErrIfRangeMismatch) {
					// If the If-Range is mismatched we just move on to send the full 200 cached response.
				} else if errors.Is(err, ErrRangeNotSatisfiable) {
					return err
				} else {
					r.WriteError("error handling Range request", http.StatusInternalServerError)
					return err
				}
			} else {
				return nil
			}
		}

		r.SetHeaders(fetched.Cached.Entry.Metadata.Object.Header)
		r.SetHeader("Accept-Ranges", "bytes")
		r.SetHeader("ETag", fetched.Cached.Entry.Metadata.Object.ETag)
		r.SetHeader("Last-Modified", fetched.Cached.Entry.Metadata.Object.LastModified.Format(http.TimeFormat))
		addCacheHeaders(r, req, typeutils.Some(fetched.Cached.Entry), fetchResultToCacheStatus(fetched))

		slog.Info("Serving cached response", "url", req.URL, "key", key)
		return finalizeAndRespond(r, fetched.Cached.Entry.Data, http.StatusOK, req)

	default:
		// This should not be possible
		return fmt.Errorf("unknown fetch type: %v", fetched.Type)
	}
}

func (p *Proxy) handleHTTP(r responder.Responder, proxyReq *http.Request) error {
	slog.Info("Handling HTTP request", "host", proxyReq.Host, "remote_addr", proxyReq.RemoteAddr)
	metrics.Global.Requests.HTTPProxyRequests.Increment()

	clientHd := headers.ParseHeaderDirective(proxyReq.Header)
	clientHd.StripRegularConditionals(proxyReq.Header)

	key := cache.MakeFromRequest(proxyReq)

	return p.processRequest(r, proxyReq, key, clientHd)
}

func (p *Proxy) handleCONNECT(r responder.Responder, proxyReq *http.Request) error {
	slog.Info("Handling CONNECT request", "url", proxyReq.URL, "remote_addr", proxyReq.RemoteAddr)

	metrics.Global.Requests.HTTPSProxyRequests.Increment()

	clientConn, _, err := r.Hijack()
	if err != nil {
		r.WriteError("Unable to take over socket.", http.StatusInternalServerError)
		return err
	}
	defer clientConn.Close() // Ensure we always close the hijacked connection

	intermediateResponder := responder.NewRawHTTPResponder(clientConn)

	tlsCert, err := p.ca.GetCertForHost(proxyReq.Host)
	if err != nil {
		// can't use http.Error after hijacking, so we write directly
		slog.Error("Error getting TLS certificate", "host", proxyReq.Host, "error", err)
		intermediateResponder.WriteError("Error getting TLS certificate", http.StatusInternalServerError)
		return fmt.Errorf("%w: %v", ErrTLSCertFailed, err)
	}

	// Send an HTTP OK response back to the client. This initiates the CONNECT
	// tunnel. From this point on the client will assume it's connected directly
	// to the target.
	if err := intermediateResponder.WriteEmpty(http.StatusOK); err != nil {
		slog.Error("Failed to write HTTP OK response to client", "error", err)
		return fmt.Errorf("%w: %v", ErrClientResponseFailed, err)
	}
	slog.Debug("Sent HTTP 200 OK response to client, established CONNECT tunnel")

	// Configure a new TLS server, pointing it at the client connection, using
	// our certificate. This server will now pretend being the target.
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{*tlsCert},
	}
	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()

	// Create a buffered reader for the client connection. This is required to
	// use http package functions with this connection.
	connReader := bufio.NewReader(tlsConn)
	responder := responder.NewRawHTTPResponder(tlsConn)

	slog.Debug("Entering request loop for CONNECT tunnel")
	for {
		// Read next HTTP request from client.
		req, err := http.ReadRequest(connReader)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			slog.Error("Error reading request from client", "remote_addr", proxyReq.RemoteAddr, "error", err)
			return fmt.Errorf("%w: %v ", ErrReadRequestFailed, err)
		}

		if err := p.handleHTTP(responder, req); err != nil {
			slog.Error("Error processing HTTP request", "remote_addr", proxyReq.RemoteAddr, "error", err)
		}
	}

	return nil
}
