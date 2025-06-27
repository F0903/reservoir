package proxy

import (
	"io"
	"net/http"
	"strings"
)

func makeHTTPResponseWithStream(status int, stream io.ReadCloser, header http.Header) *http.Response {
	return &http.Response{
		StatusCode: status,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Body:       stream,
	}
}

func makeHTTPResponseWithString(status int, text string, header http.Header) *http.Response {
	return makeHTTPResponseWithStream(status, io.NopCloser(strings.NewReader(text)), header)
}

func makeHTTPErrorResponse(err error) *http.Response {
	return makeHTTPResponseWithString(http.StatusInternalServerError, err.Error(), make(http.Header))
}
