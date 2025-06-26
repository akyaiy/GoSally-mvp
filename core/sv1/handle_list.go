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
	log := h.log.With(
		slog.Group("request",
			slog.String("version", h.GetVersion()),
			slog.String("url", h.r.URL.String()),
			slog.String("method", h.r.Method),
		),
		slog.Group("connection",
			slog.String("connection-uuid", uuid16),
			slog.String("remote", h.r.RemoteAddr),
		),
	)
	log.Info("Received request")
	type ComMeta struct {
		Description string            `json:"Description"`
		Arguments   map[string]string `json:"Arguments,omitempty"`
	}
	var (
		files         []os.DirEntry
		err           error
		commands      = make(map[string]ComMeta)
		cmdsProcessed = make(map[string]bool)
	)

	if files, err = os.ReadDir(h.cfg.ComDir); err != nil {
		log.Error("Failed to read commands directory",
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

	log.Debug("Command list prepared")

	log.Info("Session completed")

	h.w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(h.w).Encode(commands)
}
