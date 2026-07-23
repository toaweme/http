package server

import (
	"net/http"
	"strconv"
	"strings"
)

// CorsConfig controls the CrossOrigin middleware. An empty AllowedOrigins
// blocks every cross-origin request, so callers must opt in explicitly.
type CorsConfig struct {
	// AllowedOrigins lists the origins a browser may send credentialed
	// requests from. A single "*" allows any origin; with AllowCredentials
	// the matched origin is echoed back rather than "*", as the spec requires.
	AllowedOrigins []string
	// AllowedMethods lists the methods advertised on a preflight response.
	AllowedMethods []string
	// AllowedHeaders lists the request headers advertised on a preflight response.
	AllowedHeaders []string
	// ExposedHeaders lists response headers the browser may read.
	ExposedHeaders []string
	// AllowCredentials sets Access-Control-Allow-Credentials on the response.
	AllowCredentials bool
	// MaxAge caps how long, in seconds, a browser may cache a preflight result.
	MaxAge int
}

// CrossOrigin returns a middleware that applies cfg as the CORS policy. It sets
// the Access-Control-Allow-* headers on allowed cross-origin requests and
// answers preflight OPTIONS requests directly with 204, short-circuiting the
// router so they never reach a route handler.
func CrossOrigin(cfg CorsConfig) func(http.Handler) http.Handler {
	allowAll := len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*"
	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	exposed := strings.Join(cfg.ExposedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// same-origin and non-browser requests carry no Origin, so there is
			// nothing to negotiate; hand them straight to the router.
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Add("Vary", "Origin")

			if !allowAll && !originAllowed(cfg.AllowedOrigins, origin) {
				// a preflight for a disallowed origin is still a preflight; end
				// it here without CORS headers rather than routing an OPTIONS.
				if isPreflight(r) {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// with credentials the spec forbids the "*" wildcard, so echo the
			// concrete origin; without credentials "*" is fine to pass through.
			if allowAll && !cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if isPreflight(r) {
				if methods != "" {
					w.Header().Set("Access-Control-Allow-Methods", methods)
				}
				if headers != "" {
					w.Header().Set("Access-Control-Allow-Headers", headers)
				}
				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
				w.Header().Add("Vary", "Access-Control-Request-Method")
				w.Header().Add("Vary", "Access-Control-Request-Headers")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if exposed != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposed)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// originAllowed reports whether origin exactly matches one of the allowed
// origins. Matching is case-insensitive on the scheme and host per RFC 6454.
func originAllowed(allowed []string, origin string) bool {
	for _, a := range allowed {
		if strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}

// isPreflight reports whether r is a CORS preflight: an OPTIONS request that
// carries the Access-Control-Request-Method header the browser adds.
func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}
