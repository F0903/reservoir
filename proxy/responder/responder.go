package responder

import (
	"io"
	"net/http"
)

type Responder interface {
	// Sets the HTTP headers for the response.
	SetHeader(header http.Header)

	// Retrieves the HTTP headers for the response.
	GetHeader() http.Header

	// Writes the response with the given status code and body.
	Write(status int, body io.Reader) (written int64, err error)

	// Writes an empty response with the given status code.
	// Useful for responses that do not require a body.
	WriteEmpty(status int) error

	// Writes an error response with the given error message and status code.
	WriteError(message string, errorCode int) error
}
