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
	uuid16 := newUUID()
	_log.Info("Received request", slog.String("version", "v1"), slog.String("connection-uuid", uuid16), slog.String("remote", r.RemoteAddr), slog.String("method", r.Method), slog.String("url", r.URL.String()))
	type ComMeta struct {
		Description string
	}
	var (
		files    []os.DirEntry
		err      error
		com      ComMeta
		commands = make(map[string]ComMeta)
	)

	if files, err = os.ReadDir(cfg.ComDir); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to read commands directory: "+err.Error())
		_log.Error("Failed to read commands directory", slog.String("error", err.Error()))
		return
	}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".lua" {
			continue
		}
		cmdName := file.Name()[:len(file.Name())-4] // remove .lua extension
		if !allowedCmd.MatchString(string([]rune(cmdName)[0])) {
			continue
		}
		if !listAllowedCmd.MatchString(cmdName) {
			continue
		}
		if com.Description, err = extractDescriptionStatic(filepath.Join(cfg.ComDir, file.Name())); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to read command: "+err.Error())
			log.Error("Failed to read command", slog.String("error", err.Error()))
			return
		}
		if com.Description == "" {
			com.Description = "description missing"
		}
		commands[cmdName] = ComMeta{Description: com.Description}
	}
	json.NewEncoder(w).Encode(commands)
	_log.Info("Command executed successfully", slog.String("connection-uuid", uuid16))
	_log.Info("Session completed", slog.String("connection-uuid", uuid16), slog.String("remote", r.RemoteAddr), slog.String("method", r.Method), slog.String("url", r.URL.String()))
}
