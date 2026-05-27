package server

import (
	"encoding/json"
	"net/http"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiErrorResponse struct {
	Error apiError `json:"error"`
}

// writeJSON serialises v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeError writes a structured JSON error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiErrorResponse{Error: apiError{Code: code, Message: message}})
}
