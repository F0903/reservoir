package responder

import (
	"io"
	"net/http"
)

type HTTPResponder struct {
	writer http.ResponseWriter
}

func NewHTTPResponder(w http.ResponseWriter) *HTTPResponder {
	return &HTTPResponder{
		writer: w,
	}
}

func (c *HTTPResponder) SetHeader(header http.Header) {
	respHeader := c.writer.Header()
	for key, values := range header {
		for _, value := range values {
			respHeader.Add(key, value)
		}
	}
}

func (c *HTTPResponder) writeStatusHeader(status int) {
	if status != http.StatusOK {
		c.writer.WriteHeader(status)
	}
}

func (c *HTTPResponder) Write(status int, body io.ReadCloser) error {
	defer body.Close() // Ensure the body is closed after writing
	c.writeStatusHeader(status)
	_, err := io.Copy(c.writer, body)
	return err
}

func (c *HTTPResponder) WriteEmpty(status int) error {
	c.writeStatusHeader(status)
	_, err := io.Copy(c.writer, http.NoBody)
	return err
}

func (c *HTTPResponder) Error(err error, errorCode int) {
	http.Error(c.writer, err.Error(), errorCode)
}
