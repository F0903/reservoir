package responder

import (
	"io"
	"net/http"
)

type Responder interface {
	// SetHeader sets the HTTP headers for the response.
	SetHeader(header http.Header)

	// Write writes the response with the given status code and body.
	Write(status int, body io.Reader) error

	// WriteEmpty writes an empty response with the given status code.
	// It is useful for responses that do not require a body.
	WriteEmpty(status int) error

	// Error writes an error response with the given error and status code.
	Error(err error, errorCode int)
}
