package responder

import (
	"io"
	"net/http"
)

type Responder interface {
	// Set a single HTTP header for the response.
	SetHeader(name string, value string)

	// Set multiple HTTP headers for the response.
	SetHeaders(headers http.Header)

	// Writes the response with the given status code and body.
	Write(status int, body io.Reader) (written int64, err error)

	// Writes an empty response with the given status code.
	// Useful for responses that do not require a body.
	WriteEmpty(status int) error

	// Writes an error response with the given error message and status code.
	WriteError(message string, errorCode int) error
}
