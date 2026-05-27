package handler

import (
	"encoding/json"
	"net/http"
)

// apiError is the canonical error object in every error response body.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// apiErrorResponse wraps apiError as the top-level JSON envelope.
type apiErrorResponse struct {
	Error apiError `json:"error"`
}

// writeJSON serialises v as JSON with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeError writes a JSON error response with the given code and message.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiErrorResponse{Error: apiError{Code: code, Message: message}})
}
