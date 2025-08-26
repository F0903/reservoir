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

func (c *HTTPResponder) SetHeader(name string, value string) {
	c.writer.Header().Set(name, value)
}

func (c *HTTPResponder) SetHeaders(headers http.Header) {
	for key, values := range headers {
		for _, value := range values {
			c.SetHeader(key, value)
		}
	}
}

func (c *HTTPResponder) GetHeader() http.Header {
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
