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

func (c *HTTPResponder) Write(body io.Reader) error {
	c.writer.WriteHeader(200)
	_, err := io.Copy(c.writer, body)
	return err
}

func (c *HTTPResponder) Error(err error, errorCode int) {
	http.Error(c.writer, err.Error(), errorCode)
}
