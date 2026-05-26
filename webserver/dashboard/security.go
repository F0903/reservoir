package dashboard

import (
	"net/http"

	"reservoir/webserver/dashboard/csp"
)

func secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respHeaders := w.Header()
		respHeaders.Set("Cross-Origin-Opener-Policy", "same-origin")
		respHeaders.Set("Cross-Origin-Embedder-Policy", "require-corp")
		respHeaders.Set("Cross-Origin-Resource-Policy", "same-origin")
		respHeaders.Set("Content-Security-Policy", csp.Header)

		next.ServeHTTP(w, r)
	})
}
