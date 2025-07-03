package sv1

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"regexp"

	"github.com/akyaiy/GoSally-mvp/core/config"
)

func (h *HandlerV1) ErrNotFound(w http.ResponseWriter, r *http.Request) {
	h.w = w
	h.r = r
	h._errNotFound()
}

func (h *HandlerV1) newUUID() string {
	bytes := make([]byte, int(config.GetInternalConsts().GetUUIDLength()/2))
	_, err := rand.Read(bytes)
	if err != nil {
		h.log.Error("Failed to generate UUID", slog.String("error", err.Error()))
		return ""
	}
	return hex.EncodeToString(bytes)
}

func (h *HandlerV1) _errNotFound() {
	h.writeJSONError(http.StatusBadRequest, "invalid request")
	h.log.Error("HTTP request error",
		slog.String("remote", h.r.RemoteAddr),
		slog.String("method", h.r.Method),
		slog.String("url", h.r.URL.String()),
		slog.Int("status", http.StatusBadRequest))
}

func (h *HandlerV1) writeJSONError(status int, msg string) {
	h.w.Header().Set("Content-Type", "application/json")
	h.w.WriteHeader(status)
	resp := map[string]interface{}{
		"status": "error",
		"error":  msg,
		"code":   status,
	}
	if err := json.NewEncoder(h.w).Encode(resp); err != nil {
		h.log.Error("Failed to write JSON error response",
			slog.String("error", err.Error()),
			slog.Int("status", status))
		return
	}
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

func (h *HandlerV1) comMatch(ver string, comName string) string {
	files, err := os.ReadDir(h.cfg.ComDir)
	if err != nil {
		h.log.Error("Failed to read com dir",
			slog.String("error", err.Error()))
		return ""
	}

	baseName := comName + ".lua"
	verName := comName + "?" + ver + ".lua"

	var baseFileFound string

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fname := f.Name()

		if fname == verName {
			return fname
		}

		if fname == baseName {
			baseFileFound = fname
		}
	}

	return baseFileFound
}
