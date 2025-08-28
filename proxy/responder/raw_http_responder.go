package responder

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"reservoir/utils/countingreader"
	"strconv"
	"strings"
)

// A responder that manually constructs and writes HTTP responses.
// Used by the CONNECT handler with HTTPS, since it works on a raw TCP connection.
type RawHTTPResponder struct {
	writer   io.Writer
	response *http.Response
}

func NewRawHTTPResponder(writer io.Writer) *RawHTTPResponder {
	return &RawHTTPResponder{
		writer: writer,
		response: &http.Response{
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
		},
	}
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
		for _, value := range values {
			c.SetHeader(key, value)
		}
	}
}

func (c *RawHTTPResponder) GetHeaders() http.Header {
	return c.response.Header
}

func (c *RawHTTPResponder) writeResponse() error {
	// If Content-Length is unknown, we must either use chunked encoding or close the connection.
	if c.response.ContentLength < 0 {
		c.response.TransferEncoding = []string{"chunked"}
	}

	if err := c.response.Write(c.writer); err != nil {
		return err
	}
	if buf, ok := c.writer.(*bufio.Writer); ok {
		buf.Flush()
	}

	return nil
}

func (c *RawHTTPResponder) Write(status int, body io.Reader) (written int64, err error) {
	resp := c.response

	var read int
	resp.Body = io.NopCloser(countingreader.New(body, &read))
	resp.StatusCode = status
	c.parseAndSetContentLength()

	return int64(read), c.writeResponse()
}

func (c *RawHTTPResponder) WriteEmpty(status int) error {
	resp := c.response
	resp.Body = http.NoBody
	resp.StatusCode = status

	h := resp.Header
	h.Set("Content-Length", "0") // Explicitly set Content-Length to 0 for empty responses
	resp.ContentLength = 0

	return c.writeResponse()
}

func (c *RawHTTPResponder) WriteError(message string, errorCode int) error {
	resp := c.response
	resp.StatusCode = errorCode
	resp.Body = io.NopCloser(strings.NewReader(message))
	resp.ContentLength = int64(len(message))

	h := resp.Header
	h.Set("Content-Type", "text/plain; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")

	return c.writeResponse()
}

func (c *RawHTTPResponder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, ErrHijackNotSupported
}
