package sv1

import (
	"log/slog"
	"os"
)

// func (h *HandlerV1) errNotFound(w http.ResponseWriter, r *http.Request) {
// 	utils.WriteJSONError(h.w, http.StatusBadRequest, "invalid request")
// 	h.log.Error("HTTP request error",
// 		slog.String("remote", h.r.RemoteAddr),
// 		slog.String("method", h.r.Method),
// 		slog.String("url", h.r.URL.String()),
// 		slog.Int("status", http.StatusBadRequest))
// }

// func (h *HandlerV1) extractDescriptionStatic(path string) (string, error) {
// 	data, err := os.ReadFile(path)
// 	if err != nil {
// 		return "", err
// 	}

// 	re := regexp.MustCompile(`---\s*#description\s*=\s*"([^"]+)"`)
// 	m := re.FindStringSubmatch(string(data))
// 	if len(m) <= 0 {
// 		return "", nil
// 	}
// 	return m[1], nil
// }

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
