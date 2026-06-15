// Package server provides the HTTP server, router, middleware, and auth
// primitives.
package server

import (
	"context"
	"errors"
	"strings"
)

// ErrUnauthorized is returned by Authorizer.Authorize when the caller lacks
// permission. Map to HTTP 403 in handlers.
var ErrUnauthorized = errors.New("unauthorized")

// Action names the operation being checked by an Authorizer.
type Action string

// The set of actions an Authorizer can be asked to check.
const (
	ActionRead   Action = "read"
	ActionWrite  Action = "write"
	ActionDelete Action = "delete"
	ActionAdmin  Action = "admin"
)

// Authorizer is called before privileged operations. Return nil to allow,
// ErrUnauthorized to deny, or any other error to signal an infrastructure failure.
//
// resourceID identifies the target resource; it may be empty for collection-level
// operations. Read identity from ctx via OrgIDFromContext / UserIDFromContext.
type Authorizer interface {
	Authorize(ctx context.Context, action Action, resourceID string) error
}

type contextKey uint8

const (
	ctxOrgID contextKey = iota
	ctxUserID
	ctxScopes
)

// ContextWithOrgID returns a copy of ctx carrying the org ID.
func ContextWithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, ctxOrgID, orgID)
}

// ContextWithUserID returns a copy of ctx carrying the user ID.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxUserID, userID)
}

// ContextWithScopes returns a copy of ctx carrying the caller's scopes.
func ContextWithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, ctxScopes, scopes)
}

// OrgIDFromContext reads the org ID set by ContextWithOrgID. ok is false if unset.
func OrgIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxOrgID).(string)
	return v, ok
}

// UserIDFromContext reads the user ID set by ContextWithUserID. ok is false if unset.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUserID).(string)
	return v, ok
}

// ScopesFromContext reads the scopes set by ContextWithScopes. Returns nil if unset.
func ScopesFromContext(ctx context.Context) []string {
	v, _ := ctx.Value(ctxScopes).([]string)
	return v
}

// Claims holds the identity fields extracted from a Bearer token.
type Claims struct {
	OrgID  string
	UserID string
	Scopes []string
}

// ClaimsExtractor parses a raw Bearer token and returns the claims.
// Return a non-nil error to reject the request with HTTP 401.
type ClaimsExtractor func(token string) (*Claims, error)

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	tok := header[len(prefix):]
	if tok == "" {
		return "", false
	}
	return tok, true
}
