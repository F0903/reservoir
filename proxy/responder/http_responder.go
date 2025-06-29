package responder

import (
	"io"
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

func (c *HTTPResponder) Write(status int, body io.Reader) error {
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
