package apihttp

import (
	"encoding/json"
	"log/slog"
	"mime"
	"net/http"
	"strings"
)

const JSONContentType = "application/json"

func WriteJSON(w http.ResponseWriter, status int, value any) bool {
	body, err := json.Marshal(value)
	if err != nil {
		slog.Error("Error marshaling JSON response", "error", err)
		InternalServerError(w)
		return false
	}

	w.Header().Set("Content-Type", JSONContentType)
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		slog.Error("Error writing JSON response", "error", err)
		return false
	}
	return true
}

func WriteText(w http.ResponseWriter, status int, value string) bool {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(value)); err != nil {
		slog.Error("Error writing text response", "error", err)
		return false
	}
	return true
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, message string, status int) {
	http.Error(w, message, status)
}

func BadRequest(w http.ResponseWriter, message string) {
	Error(w, message, http.StatusBadRequest)
}

func InternalServerError(w http.ResponseWriter) {
	Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func RequireJSONContentType(w http.ResponseWriter, r *http.Request) bool {
	if IsJSONContentType(r.Header.Get("Content-Type")) {
		return true
	}

	Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
	return false
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		slog.Error("Error decoding JSON request", "error", err)
		BadRequest(w, "Invalid JSON")
		return false
	}
	return true
}

func IsJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}

	mediaType = strings.ToLower(mediaType)
	return mediaType == JSONContentType || (strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json"))
}
