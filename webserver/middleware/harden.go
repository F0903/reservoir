package middleware

import (
	"log/slog"
	"net/http"
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
		respHeaders.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; font-src 'self'; worker-src 'self'; style-src 'self'; img-src 'self' blob: data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
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
