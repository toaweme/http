package server

import (
	"errors"
	"net/http"

	"github.com/toaweme/log"
)

// AuthMiddleware extracts the Bearer token, runs extract to parse it into
// Claims, injects OrgID/UserID/Scopes into the request context, and aborts
// with HTTP 401 on missing/invalid header or extractor failure.
func AuthMiddleware(extract ClaimsExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				log.Warn("server", "auth", "error", "missing authorization header")
				WriteError(w, http.StatusUnauthorized, errors.New("missing authorization header"))
				return
			}

			claims, err := extract(token)
			if err != nil || claims == nil {
				msg := "invalid authorization header claims"
				if err != nil {
					msg = msg + ": " + err.Error()
				}
				log.Warn("server", "auth", "error", msg)
				WriteError(w, http.StatusUnauthorized, errors.New(msg))
				return
			}

			ctx := WithOrgID(r.Context(), claims.OrgID)
			ctx = WithUserID(ctx, claims.UserID)
			ctx = WithScopes(ctx, claims.Scopes)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
