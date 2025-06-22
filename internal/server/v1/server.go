package server_v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"regexp"

	"GoSally-mvp/internal/config"
)

type ServerV1Contract interface {
	Handle(w http.ResponseWriter, r *http.Request)
	HandleList(w http.ResponseWriter, r *http.Request)

	_handle()
	_handleList()
}

type HandlerV1 struct {
	w http.ResponseWriter
	r *http.Request

	_log slog.Logger

	cfg *config.ConfigConf

	allowedCmd *regexp.Regexp
	listAllowedCmd *regexp.Regexp
}

func (h *HandlerV1) Handle(w http.ResponseWriter, r *http.Request) {
	h.w = w
	h.r = r
	h._handle()
}

func (h *HandlerV1) HandleList(w http.ResponseWriter, r *http.Request) {
	h.w = w
	h.r = r
	h._handleList()
}

func errNotFound(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, http.StatusBadRequest, "invalid request")
	_log.Error("HTTP request error", slog.String("remote", r.RemoteAddr), slog.String("method", r.Method), slog.String("url", r.URL.String()), slog.Int("status", http.StatusBadRequest))
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]interface{}{
		"status": "error",
		"error":  msg,
		"code":   status,
	}
	json.NewEncoder(w).Encode(resp)
}

func extractDescriptionStatic(path string) (string, error) {
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
