package proxy

import (
	"io"
	"net/http"
	"strconv"
	"strings"
)

func makeHTTPResponseWithStream(status int, stream io.ReadCloser, header http.Header) *http.Response {
	contentLen, err := strconv.ParseInt(header.Get("Content-Length"), 10, 64)
	if err != nil {
		contentLen = -1 // If Content-Length is not set or invalid, use -1 to indicate unknown length
	}

	return &http.Response{
		StatusCode:    status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          stream,
		ContentLength: contentLen,
	}
}

func makeHTTPResponseWithString(status int, text string, header http.Header) *http.Response {
	return makeHTTPResponseWithStream(status, io.NopCloser(strings.NewReader(text)), header)
}

func makeHTTPErrorResponse(err error) *http.Response {
	return makeHTTPResponseWithString(http.StatusInternalServerError, err.Error(), make(http.Header))
}
