package sv1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *HandlerV1) _handleList() {
	uuid16 := h.newUUID()
	h.log.Info("Received request",
		slog.String("version", "v1"),
		slog.String("connection-uuid", uuid16),
		slog.String("remote", h.r.RemoteAddr),
		slog.String("method", h.r.Method),
		slog.String("url", h.r.URL.String()))

	type ComMeta struct {
		Description string
	}

	var (
		files         []os.DirEntry
		err           error
		commands      = make(map[string]ComMeta)
		cmdsProcessed = make(map[string]bool)
	)

	if files, err = os.ReadDir(h.cfg.ComDir); err != nil {
		h.log.Error("Failed to read commands directory",
			slog.String("error", err.Error()))
		h.writeJSONError(http.StatusInternalServerError, "failed to read commands directory: "+err.Error())
		return
	}

	apiVer := chi.URLParam(h.r, "ver")

	// Сначала ищем версионные
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".lua" {
			continue
		}
		cmdFull := file.Name()[:len(file.Name())-4]
		cmdParts := strings.SplitN(cmdFull, "?", 2)
		cmdName := cmdParts[0]

		if !h.allowedCmd.MatchString(string([]rune(cmdName)[0])) {
			continue
		}
		if !h.listAllowedCmd.MatchString(cmdName) {
			continue
		}

		if len(cmdParts) == 2 && cmdParts[1] == apiVer {
			description, _ := h.extractDescriptionStatic(filepath.Join(h.cfg.ComDir, file.Name()))
			if description == "" {
				description = "description missing"
			}
			commands[cmdName] = ComMeta{Description: description}
			cmdsProcessed[cmdName] = true
		}
	}

	// Потом фоллбеки
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".lua" {
			continue
		}
		cmdFull := file.Name()[:len(file.Name())-4]
		cmdParts := strings.SplitN(cmdFull, "?", 2)
		cmdName := cmdParts[0]

		if !h.allowedCmd.MatchString(string([]rune(cmdName)[0])) {
			continue
		}
		if !h.listAllowedCmd.MatchString(cmdName) {
			continue
		}
		if cmdsProcessed[cmdName] {
			continue
		}
		if len(cmdParts) == 1 {
			description, _ := h.extractDescriptionStatic(filepath.Join(h.cfg.ComDir, file.Name()))
			if description == "" {
				description = "description missing"
			}
			commands[cmdName] = ComMeta{Description: description}
			cmdsProcessed[cmdName] = true
		}
	}

	h.log.Info("Command list prepared",
		slog.String("connection-uuid", uuid16))

	h.log.Info("Session completed",
		slog.String("connection-uuid", uuid16),
		slog.String("remote", h.r.RemoteAddr),
		slog.String("method", h.r.Method),
		slog.String("url", h.r.URL.String()))

	h.w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(h.w).Encode(commands)
}
