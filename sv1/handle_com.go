package sv1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	lua "github.com/yuin/gopher-lua"
)

func (h *HandlerV1) _handle() {
	uuid16 := h.newUUID()
	h.log.Info("Received request",
		slog.String("version", "v1"),
		slog.String("connection-uuid", uuid16),
		slog.String("remote", h.r.RemoteAddr),
		slog.String("method", h.r.Method),
		slog.String("url", h.r.URL.String()))

	cmd := chi.URLParam(h.r, "cmd")
	var scriptPath string
	if !h.allowedCmd.MatchString(string([]rune(cmd)[0])) {
		h.log.Error("HTTP request error",
			slog.String("connection-uuid", uuid16),
			slog.String("error", "invalid command"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusBadRequest))
		h.writeJSONError(http.StatusBadRequest, "invalid command")
		return
	}
	if !h.listAllowedCmd.MatchString(cmd) {
		h.log.Error("HTTP request error",
			slog.String("connection-uuid", uuid16),
			slog.String("error", "invalid command"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusBadRequest))
		h.writeJSONError(http.StatusBadRequest, "invalid command")
		return
	}
	if scriptPath = h.comMatch(chi.URLParam(h.r, "ver"), cmd); scriptPath == "" {
		h.log.Error("HTTP request error",
			slog.String("connection-uuid", uuid16),
			slog.String("error", "command not found"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusNotFound))
		h.writeJSONError(http.StatusNotFound, "command not found")
		return
	}

	scriptPath = filepath.Join(h.cfg.ComDir, scriptPath)
	if _, err := os.Stat(scriptPath); err != nil {
		h.log.Error("HTTP request error",
			slog.String("connection-uuid", uuid16),
			slog.String("error", "command not found"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusNotFound))
		h.writeJSONError(http.StatusNotFound, "command not found")
		return
	}

	L := lua.NewState()
	defer L.Close()

	L.OpenLibs() // loads base, io, os, string, math, table, debug, package, coroutine, channel… :contentReference[oaicite:0]{index=0}

	qt := h.r.URL.Query()
	tbl := L.NewTable()
	for k, v := range qt {
		if len(v) > 0 {
			L.SetField(tbl, k, lua.LString(v[0]))
		}
	}
	L.SetGlobal("Params", tbl)
	L.SetGlobal("Result", L.NewTable())

	prepareLuaEnv := filepath.Join(h.cfg.ComDir, "_prepare"+".lua")
	if _, err := os.Stat(prepareLuaEnv); err == nil {
		if err := L.DoFile(prepareLuaEnv); err != nil {
			h.log.Error("Failed to prepare lua environment",
				slog.String("connection-uuid", uuid16),
				slog.String("error", err.Error()))
			h.writeJSONError(http.StatusInternalServerError, "lua error: "+err.Error())
			return
		}
	} else {
		h.log.Error("No environment preparation script found, skipping preparation", slog.String("connection-uuid", uuid16), slog.String("error", err.Error()))
	}

	if err := L.DoFile(scriptPath); err != nil {
		h.log.Error("Failed to execute lua script",
			slog.String("connection-uuid", uuid16),
			slog.String("error", err.Error()))
		h.writeJSONError(http.StatusInternalServerError, "lua error: "+err.Error())
		return
	}

	out := make(map[string]any)
	if rt := L.GetGlobal("Result"); rt.Type() == lua.LTTable {
		rt.(*lua.LTable).ForEach(func(k, v lua.LValue) {
			switch v.Type() {
			case lua.LTString:
				out[k.String()] = v.String()
			case lua.LTNumber:
				out[k.String()] = float64(v.(lua.LNumber))
			case lua.LTBool:
				out[k.String()] = bool(v.(lua.LBool))
			default:
				out[k.String()] = v.String()
			}
		})
	}

	h.w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(h.w).Encode(out)
	switch out["status"] {
	case "error":
		h.log.Info("Command executed with error",
			slog.String("connection-uuid", uuid16),
			slog.String("cmd", cmd),
			slog.Any("result", out))
	case "ok":
		h.log.Info("Command executed successfully",
			slog.String("connection-uuid", uuid16),
			slog.String("cmd", cmd), slog.Any("result", out))
	default:
		h.log.Info("Command executed and returned an unknown status",
			slog.String("connection-uuid", uuid16),
			slog.String("cmd", cmd),
			slog.Any("result", out))
	}
	h.log.Info("Session completed",
		slog.String("connection-uuid", uuid16),
		slog.String("remote", h.r.RemoteAddr),
		slog.String("method", h.r.Method),
		slog.String("url", h.r.URL.String()))
}
