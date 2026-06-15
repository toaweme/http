package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_WriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, map[string]string{"hello": "world"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type: got %q want application/json", ct)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"hello":"world"}` {
		t.Fatalf("body: got %q", got)
	}
}

func Test_WriteError(t *testing.T) {
	tests := []struct {
		name       string
		write      func(w http.ResponseWriter)
		wantStatus int
	}{
		{
			name:       "WriteError uses given status",
			write:      func(w http.ResponseWriter) { WriteError(w, http.StatusForbidden, errors.New("nope")) },
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "WriteBadRequest is 400",
			write:      func(w http.ResponseWriter) { WriteBadRequest(w, errors.New("bad input")) },
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.write(rec)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status: got %d want %d", rec.Code, tt.wantStatus)
			}
			var body ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if body.Error == "" {
				t.Fatalf("error field empty, body=%q", rec.Body.String())
			}
		})
	}
}

func Test_ReadJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	t.Run("decodes valid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ada"}`))
		var p payload
		if err := ReadJSON(req, &p); err != nil {
			t.Fatalf("ReadJSON: %v", err)
		}
		if p.Name != "ada" {
			t.Fatalf("name: got %q want ada", p.Name)
		}
	})

	t.Run("errors on malformed body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{not json`))
		var p payload
		if err := ReadJSON(req, &p); err == nil {
			t.Fatal("expected error on malformed JSON, got nil")
		}
	})
}

func Test_ReadRawJSON(t *testing.T) {
	t.Run("returns raw message for valid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":1}`))
		raw, err := ReadRawJSON(req)
		if err != nil {
			t.Fatalf("ReadRawJSON: %v", err)
		}
		if string(raw) != `{"a":1}` {
			t.Fatalf("raw: got %q want {\"a\":1}", string(raw))
		}
	})

	t.Run("errors on malformed JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`}{`))
		if _, err := ReadRawJSON(req); err == nil {
			t.Fatal("expected error on malformed JSON, got nil")
		}
	})
}
