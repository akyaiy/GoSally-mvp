package sv1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/akyaiy/GoSally-mvp/core/config"
	"github.com/akyaiy/GoSally-mvp/core/corestate"
	"github.com/akyaiy/GoSally-mvp/core/utils"
	"github.com/go-chi/chi/v5"
)

// The function processes the HTTP request and returns a list of available commands.
func (h *HandlerV1) HandleList(w http.ResponseWriter, r *http.Request) {
	uuid16, err := utils.NewUUID(int(config.UUIDLength))
	if err != nil {
		h.log.Error("Failed to generate UUID",
			slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to generate UUID: "+err.Error())
		return
	}
	log := h.log.With(
		slog.Group("request",
			slog.String("version", h.GetVersion()),
			slog.String("url", r.URL.String()),
			slog.String("method", r.Method),
		),
		slog.Group("connection",
			slog.String("connection-uuid", uuid16),
			slog.String("remote", r.RemoteAddr),
		),
	)
	log.Info("Received request")
	type ComMeta struct {
		Description string            `json:"Description"`
		Arguments   map[string]string `json:"Arguments,omitempty"`
	}
	var (
		files         []os.DirEntry
		commands      = make(map[string]ComMeta)
		cmdsProcessed = make(map[string]bool)
	)

	if files, err = os.ReadDir(h.cfg.ComDir); err != nil {
		log.Error("Failed to read commands directory",
			slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to read commands directory: "+err.Error())
		return
	}

	apiVer := chi.URLParam(r, "ver")

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
	uuid32, _ := corestate.GetNodeUUID(filepath.Join(config.MetaDir, "uuid"))
	response := ResponseFormat{
		ResponsibleAgentUUID: uuid32,
		RequestedCommand:     "list",
		Response:             commands,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Failed to write JSON error response",
			slog.String("error", err.Error()))
	}
}
