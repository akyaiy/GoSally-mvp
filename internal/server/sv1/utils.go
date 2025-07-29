package sv1

import (
	"log/slog"
	"os"
)

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
