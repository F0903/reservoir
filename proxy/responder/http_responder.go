package responder

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
)

// Essentially a simple wrapper around http.ResponseWriter
// Used when getting simple HTTP requests from the client
// as we can use the provided http.ResponseWriter directly
type HTTPResponder struct {
	writer http.ResponseWriter
}

func NewHTTPResponder(w http.ResponseWriter) *HTTPResponder {
	return &HTTPResponder{
		writer: w,
	}
}

func (c *HTTPResponder) SetHeader(name string, value string) {
	c.writer.Header().Set(name, value)
}

func (c *HTTPResponder) AddHeader(name string, value string) {
	c.writer.Header().Add(name, value)
}

func (c *HTTPResponder) SetHeaders(headers http.Header) {
	for key, values := range headers {
		for _, value := range values {
			c.SetHeader(key, value)
		}
	}
}

func (c *HTTPResponder) GetHeaders() http.Header {
	return c.writer.Header()
}

func (c *HTTPResponder) writeStatusHeader(status int) {
	if status != http.StatusOK {
		c.writer.WriteHeader(status)
	}
}

func (c *HTTPResponder) Write(status int, body io.Reader) (written int64, err error) {
	c.writeStatusHeader(status)
	return io.Copy(c.writer, body)
}

func (c *HTTPResponder) WriteEmpty(status int) error {
	c.writeStatusHeader(status)
	_, err := io.Copy(c.writer, http.NoBody)
	return err
}

func (c *HTTPResponder) WriteError(message string, errorCode int) error {
	http.Error(c.writer, message, errorCode)
	return nil
}

func (c *HTTPResponder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	// "Hijack" the client connection to get a TCP (or TLS) socket we can read and write arbitrary data to/from.
	hj, ok := c.writer.(http.Hijacker)
	if !ok {
		slog.Error("Could not hijack connection. Client might be connected with unsupported HTTP version", "host", c.GetHeaders().Get("Host"), "writer_type", fmt.Sprintf("%T", c.writer))
		return nil, nil, ErrHijackNotSupported
	}

	// Hijack the connection to get the underlying net.Conn.
	clientConn, _, err := hj.Hijack()
	if err != nil {
		slog.Error("Failed to hijack connection", "error", err)
		return nil, nil, fmt.Errorf("%w: %v", ErrHijackFailed, err)
	}

	return clientConn, nil, nil
}
