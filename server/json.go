package server

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the canonical JSON error body.
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteJSON encodes v as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes err as a JSON ErrorResponse with the given status code.
// Callers map domain errors to status codes themselves.
func WriteError(w http.ResponseWriter, status int, err error) {
	WriteJSON(w, status, ErrorResponse{Error: err.Error()})
}

// WriteBadRequest writes err with HTTP 400.
func WriteBadRequest(w http.ResponseWriter, err error) {
	WriteError(w, http.StatusBadRequest, err)
}

// ReadJSON decodes the request body into v.
func ReadJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// ReadRawJSON returns the request body as a json.RawMessage after validating
// it is well-formed JSON.
func ReadRawJSON(r *http.Request) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}
