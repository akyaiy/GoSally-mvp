package utils

import (
	"encoding/json"
	"net/http"
)

// writeJSONError writes a JSON error response to the HTTP response writer.
// It sets the Content-Type to application/json, writes the specified HTTP status code
func WriteJSONError(w http.ResponseWriter, status int, msg string) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]any{
		"status": "error",
		"error":  msg,
		"code":   status,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return err
	}
	return nil
}
