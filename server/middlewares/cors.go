package middlewares

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/toaweme/log"
)

// CORSConfig configures the CORS middleware. Empty slices fall back to
// permissive defaults suitable for typical JSON APIs.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// CORS returns a middleware that applies CORS headers based on cfg
// and short-circuits OPTIONS preflight requests with 204.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = []string{"Authorization", "Content-Type"}
	}
	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{"*"}
	}

	allowAll := false
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowAll = true
			break
		}
	}

	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(cfg.ExposedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			log.Info("cors", "origin", origin)
			if origin != "" {
				allowed := ""
				switch {
				case allowAll && !cfg.AllowCredentials:
					allowed = "*"
				case allowAll && cfg.AllowCredentials:
					// credentials forbid wildcard, echo the request origin
					allowed = origin
				default:
					if matchOrigin(origin, cfg.AllowedOrigins) {
						allowed = origin
					}
				}

				if allowed != "" {
					h := w.Header()
					h.Set("Access-Control-Allow-Origin", allowed)
					if allowed != "*" {
						h.Add("Vary", "Origin")
					}
					if cfg.AllowCredentials {
						h.Set("Access-Control-Allow-Credentials", "true")
					}
					if exposedHeaders != "" {
						h.Set("Access-Control-Expose-Headers", exposedHeaders)
					}

					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						h.Add("Vary", "Access-Control-Request-Method")
						h.Add("Vary", "Access-Control-Request-Headers")
						h.Set("Access-Control-Allow-Methods", allowedMethods)
						h.Set("Access-Control-Allow-Headers", allowedHeaders)
						if cfg.MaxAge > 0 {
							h.Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
						}
						w.WriteHeader(http.StatusNoContent)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func matchOrigin(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == origin {
			return true
		}
		if strings.HasPrefix(a, "*.") && strings.HasSuffix(origin, a[1:]) {
			return true
		}
	}
	return false
}
