package responder

import (
	"io"
	"net/http"
	"strconv"
	"strings"
)

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

func (c *RawHTTPResponder) Write(status int, body io.Reader) error {
	c.response.Body = io.NopCloser(body)
	c.parseAndSetContentLength()
	c.response.StatusCode = status
	return c.response.Write(c.writer)
}

func (c *RawHTTPResponder) WriteEmpty(status int) error {
	c.response.StatusCode = status
	c.response.Body = http.NoBody
	return c.response.Write(c.writer)
}

func (c *RawHTTPResponder) Error(err error, errorCode int) {
	content := err.Error()
	c.response.StatusCode = errorCode
	c.response.Body = io.NopCloser(strings.NewReader(content))
	c.response.ContentLength = int64(len(content))
}
