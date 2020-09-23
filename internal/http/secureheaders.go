package http

import "net/http"

// SecureHeaders adds security-related headers to all responses.
func SecureHeaders() Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Don't allow frame embedding.
			w.Header().Set("X-Frame-Options", "deny")
			// Prevent MIME sniffing.
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// Block cross-site scripting attacks.
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			h.ServeHTTP(w, r)
		})
	}
}
