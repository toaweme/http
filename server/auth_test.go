package server

import (
	"testing"
)

func Test_bearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{name: "valid bearer", header: "Bearer abc123", wantToken: "abc123", wantOK: true},
		{name: "missing prefix", header: "abc123", wantToken: "", wantOK: false},
		{name: "wrong scheme", header: "Basic abc123", wantToken: "", wantOK: false},
		{name: "empty token after prefix", header: "Bearer ", wantToken: "", wantOK: false},
		{name: "empty header", header: "", wantToken: "", wantOK: false},
		{name: "case sensitive scheme", header: "bearer abc", wantToken: "", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := bearerToken(tt.header)
			if got != tt.wantToken || ok != tt.wantOK {
				t.Fatalf("bearerToken(%q): got (%q, %v) want (%q, %v)", tt.header, got, ok, tt.wantToken, tt.wantOK)
			}
		})
	}
}

func Test_ContextRoundTrip(t *testing.T) {
	ctx := t.Context()
	ctx = ContextWithOrgID(ctx, "org-1")
	ctx = ContextWithUserID(ctx, "user-1")
	ctx = ContextWithScopes(ctx, []string{"read", "write"})

	if org, ok := OrgIDFromContext(ctx); !ok || org != "org-1" {
		t.Fatalf("org: got (%q, %v) want (org-1, true)", org, ok)
	}
	if user, ok := UserIDFromContext(ctx); !ok || user != "user-1" {
		t.Fatalf("user: got (%q, %v) want (user-1, true)", user, ok)
	}
	scopes := ScopesFromContext(ctx)
	if len(scopes) != 2 || scopes[0] != "read" || scopes[1] != "write" {
		t.Fatalf("scopes: got %v want [read write]", scopes)
	}
}

func Test_ContextGetters_Unset(t *testing.T) {
	ctx := t.Context()
	if org, ok := OrgIDFromContext(ctx); ok || org != "" {
		t.Fatalf("unset org: got (%q, %v) want empty,false", org, ok)
	}
	if user, ok := UserIDFromContext(ctx); ok || user != "" {
		t.Fatalf("unset user: got (%q, %v) want empty,false", user, ok)
	}
	if scopes := ScopesFromContext(ctx); scopes != nil {
		t.Fatalf("unset scopes: got %v want nil", scopes)
	}
}
