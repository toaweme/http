package server

import (
	"context"
	"errors"
	"strings"
)

// ErrUnauthorized is returned by Authorizer.Authorize when the caller lacks
// permission. Map to HTTP 403 in handlers.
var ErrUnauthorized = errors.New("server: unauthorized")

// Action names the operation being checked by an Authorizer.
type Action string

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

func WithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, ctxOrgID, orgID)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxUserID, userID)
}

func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, ctxScopes, scopes)
}

func OrgIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxOrgID).(string)
	return v, ok
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUserID).(string)
	return v, ok
}

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
