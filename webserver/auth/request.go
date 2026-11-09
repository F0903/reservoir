package auth

import (
	"net/http"
	"strings"
)

func RequestUsesHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return forwardedProtoIsHTTPS(r.Header)
}

func forwardedProtoIsHTTPS(header http.Header) bool {
	for _, value := range header.Values("Forwarded") {
		if firstForwardedProto(value) == "https" {
			return true
		}
	}
	for _, value := range header.Values("X-Forwarded-Proto") {
		if firstHeaderToken(value) == "https" {
			return true
		}
	}
	return false
}

func firstForwardedProto(value string) string {
	for group := range strings.SplitSeq(value, ",") {
		for part := range strings.SplitSeq(group, ";") {
			key, val, ok := strings.Cut(strings.TrimSpace(part), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "proto") {
				continue
			}
			return strings.ToLower(strings.Trim(strings.TrimSpace(val), `"`))
		}
	}
	return ""
}

func firstHeaderToken(value string) string {
	for token := range strings.SplitSeq(value, ",") {
		token = strings.TrimSpace(token)
		if token != "" {
			return strings.ToLower(token)
		}
	}
	return ""
}
