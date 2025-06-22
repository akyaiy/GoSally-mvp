package v1

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"regexp"
)

func (h *HandlerV1) ErrNotFound(w http.ResponseWriter, r *http.Request) {
	h.w = w
	h.r = r
	h._errNotFound()
}

func (h *HandlerV1) newUUID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		h.log.Error("Failed to generate UUID", slog.String("error", err.Error()))
		return ""
	}
	return hex.EncodeToString(bytes)
}

func (h *HandlerV1) _errNotFound() {
	h.writeJSONError(http.StatusBadRequest, "invalid request")
	h.log.Error("HTTP request error", slog.String("remote", h.r.RemoteAddr), slog.String("method", h.r.Method), slog.String("url", h.r.URL.String()), slog.Int("status", http.StatusBadRequest))
}

func (h *HandlerV1) writeJSONError(status int, msg string) {
	h.w.Header().Set("Content-Type", "application/json")
	h.w.WriteHeader(status)
	resp := map[string]interface{}{
		"status": "error",
		"error":  msg,
		"code":   status,
	}
	json.NewEncoder(h.w).Encode(resp)
}

func (h *HandlerV1) extractDescriptionStatic(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`---\s*#description\s*=\s*"([^"]+)"`)
	m := re.FindStringSubmatch(string(data))
	if len(m) <= 0 {
		return "", nil
	}
	return m[1], nil
}
