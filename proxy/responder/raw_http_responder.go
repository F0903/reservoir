package responder

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"reservoir/utils/countingreader"
	"strconv"
	"strings"
	"time"
)

// A responder that manually constructs and writes HTTP responses.
// Used by the CONNECT handler with HTTPS, since it works on a raw TCP connection.
type RawHTTPResponder struct {
	writer   io.Writer
	response *http.Response
}

func newRawResponse() *http.Response {
	return &http.Response{
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
}

func NewRawHTTPResponder(writer io.Writer) *RawHTTPResponder {
	return &RawHTTPResponder{
		writer:   writer,
		response: newRawResponse(),
	}
}

func (c *RawHTTPResponder) resetResponse() {
	c.response = newRawResponse()
}

func (c *RawHTTPResponder) parseAndSetContentLength() error {
	header := c.response.Header

	contentLenString := header.Get("Content-Length")
	contentLen, err := strconv.ParseInt(contentLenString, 10, 64)
	if err != nil {
		contentLen = -1 // If Content-Length is not set or invalid, use -1 to indicate unknown length
	}

	c.response.ContentLength = contentLen
	return nil
}

func (c *RawHTTPResponder) SetHeader(name string, value string) {
	c.response.Header.Set(name, value)
}

func (c *RawHTTPResponder) AddHeader(name string, value string) {
	c.response.Header.Add(name, value)
}

func (c *RawHTTPResponder) SetHeaders(headers http.Header) {
	for key, values := range headers {
		c.response.Header.Del(key)
		for _, value := range values {
			c.AddHeader(key, value)
		}
	}
}

func (c *RawHTTPResponder) GetHeaders() http.Header {
	return c.response.Header
}

func (c *RawHTTPResponder) writeResponse() (time.Duration, error) {
	// If Content-Length is unknown, we must either use chunked encoding or close the connection.
	if c.response.ContentLength < 0 {
		c.response.TransferEncoding = []string{"chunked"}
	}

	timed := &timedWriter{writer: c.writer}
	if err := c.response.Write(timed); err != nil {
		return timed.duration, err
	}
	if buf, ok := c.writer.(*bufio.Writer); ok {
		flushStart := time.Now()
		err := buf.Flush()
		timed.duration += time.Since(flushStart)
		if err != nil {
			return timed.duration, err
		}
	}

	return timed.duration, nil
}

func (c *RawHTTPResponder) Write(status int, body io.Reader) (written int64, writeDuration time.Duration, err error) {
	defer c.resetResponse()

	resp := c.response

	var read int
	resp.Body = io.NopCloser(countingreader.New(body, &read))
	resp.StatusCode = status
	c.parseAndSetContentLength()

	writeDuration, err = c.writeResponse()
	return int64(read), writeDuration, err
}

func (c *RawHTTPResponder) WriteEmpty(status int) error {
	defer c.resetResponse()

	resp := c.response
	resp.Body = http.NoBody
	resp.StatusCode = status

	h := resp.Header
	h.Set("Content-Length", "0") // Explicitly set Content-Length to 0 for empty responses
	resp.ContentLength = 0

	_, err := c.writeResponse()
	return err
}

func (c *RawHTTPResponder) WriteError(message string, errorCode int) error {
	defer c.resetResponse()

	resp := c.response
	resp.StatusCode = errorCode
	resp.Body = io.NopCloser(strings.NewReader(message))
	resp.ContentLength = int64(len(message))

	h := resp.Header
	h.Set("Content-Type", "text/plain; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")

	_, err := c.writeResponse()
	return err
}

func (c *RawHTTPResponder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, ErrHijackNotSupported
}
