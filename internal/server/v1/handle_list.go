package server_v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/go-chi/chi/v5"
)

func (h *HandlerV1) _handleList() {
	uuid16 := h.newUUID()
	h.log.Info("Received request", slog.String("version", "v1"), slog.String("connection-uuid", uuid16), slog.String("remote", h.r.RemoteAddr), slog.String("method", h.r.Method), slog.String("url", h.r.URL.String()))
	type ComMeta struct {
		Description string
	}
	var (
		files    []os.DirEntry
		err      error
		com      ComMeta
		commands = make(map[string]ComMeta)
	)

	if files, err = os.ReadDir(h.cfg.ComDir); err != nil {
		h.writeJSONError(http.StatusInternalServerError, "failed to read commands directory: "+err.Error())
		h.log.Error("Failed to read commands directory", slog.String("error", err.Error()))
		return
	}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".lua" {
			continue
		}
		cmdName := file.Name()[:len(file.Name())-4] // remove .lua extension
		if !h.allowedCmd.MatchString(string([]rune(cmdName)[0])) {
			continue
		}
		if !h.listAllowedCmd.MatchString(cmdName) {
			continue
		}
		if com.Description, err = h.extractDescriptionStatic(filepath.Join(h.cfg.ComDir, file.Name())); err != nil {
			h.writeJSONError(http.StatusInternalServerError, "failed to read command: "+err.Error())
			h.log.Error("Failed to read command", slog.String("error", err.Error()))
			return
		}
		if com.Description == "" {
			com.Description = "description missing"
		}
		commands[cmdName] = ComMeta{Description: com.Description}
	}
	json.NewEncoder(h.w).Encode(commands)
	h.log.Info("Command executed successfully", slog.String("connection-uuid", uuid16))
	h.log.Info("Session completed", slog.String("connection-uuid", uuid16), slog.String("remote", h.r.RemoteAddr), slog.String("method", h.r.Method), slog.String("url", h.r.URL.String()))
}
