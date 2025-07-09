package responder

import (
	"apt_cacher_go/utils/counting_reader"
	"bufio"
	"io"
	"log"
	"net/http"
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

func (c *RawHTTPResponder) SetHeader(header http.Header) {
	c.response.Header = header
}

func (c *RawHTTPResponder) writeResponse() error {
	// If Content-Length is unknown, we must either use chunked encoding or close the connection.
	if c.response.ContentLength < 0 {
		c.response.TransferEncoding = []string{"chunked"}
	}

	if err := c.response.Write(c.writer); err != nil {
		log.Printf("error writing response in RawHTTPResponder: %v", err)
		return err
	}
	if buf, ok := c.writer.(*bufio.Writer); ok {
		buf.Flush()
	}

	return nil
}

func (c *RawHTTPResponder) Write(status int, body io.Reader) (written int64, err error) {
	var read int
	c.response.Body = io.NopCloser(counting_reader.NewCountingReader(body, &read))
	c.response.StatusCode = status
	c.parseAndSetContentLength()

	return int64(read), c.writeResponse()
}

func (c *RawHTTPResponder) WriteEmpty(status int) error {
	c.response.Body = http.NoBody
	c.response.StatusCode = status

	header := c.response.Header
	header.Set("Content-Length", "0") // Explicitly set Content-Length to 0 for empty responses
	c.response.ContentLength = 0

	return c.writeResponse()
}

func (c *RawHTTPResponder) Error(message string, errorCode int) {
	c.response.StatusCode = errorCode
	c.response.Body = io.NopCloser(strings.NewReader(message))
	c.response.ContentLength = int64(len(message))
}
