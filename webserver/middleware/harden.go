package middleware

import (
	"log/slog"
	"net/http"
	"reservoir/webserver/dashboard/csp"
)

func Harden(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respHeaders := w.Header()
		respHeaders.Set("Cross-Origin-Opener-Policy", "same-origin")
		respHeaders.Set("Cross-Origin-Embedder-Policy", "require-corp")
		respHeaders.Set("Cross-Origin-Resource-Policy", "same-origin")
		respHeaders.Set("Referrer-Policy", "no-referrer")
		respHeaders.Set("X-Frame-Options", "DENY")
		respHeaders.Set("X-Content-Type-Options", "nosniff")
		respHeaders.Set("Content-Security-Policy", csp.Header /* If this isn't defined, you need to run the generator */)
		respHeaders.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), magnetometer=(), accelerometer=(), gyroscope=()")

		site := r.Header.Get("Sec-Fetch-Site")
		origin := r.Header.Get("Origin")

		isSame := origin == "" || (site == "" || site == "same-origin" || site == "same-site")
		allowed := isSame

		if !allowed {
			slog.Warn("Cross-site request blocked", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "origin", origin, "site", site)
			http.Error(w, "Cross-site request blocked", http.StatusForbidden)
			return
		}

		// Kill CORS preflight outright
		if r.Method == http.MethodOptions && origin != "" {
			slog.Warn("CORS preflight blocked", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "origin", origin, "site", site)
			http.Error(w, "CORS preflight blocked", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
