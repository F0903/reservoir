package responder

import (
	"io"
	"net/http"
)

type Responder interface {
	SetHeader(header http.Header)
	Write(body io.Reader) error
	Error(err error, errorCode int)
}
