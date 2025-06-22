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
	h.log.Info("Received request", slog.String("version", "v1"), slog.String("connection-uuid", uuid16), slog.String("remote", h.r.RemoteAddr), slog.String("method", h.r.Method), slog.String("url", h.r.URL.String()))

	cmd := chi.URLParam(h.r, "cmd")
	if !h.allowedCmd.MatchString(string([]rune(cmd)[0])) {
		h.writeJSONError(http.StatusBadRequest, "invalid command")
		h.log.Error("HTTP request error", slog.String("connection-uuid", uuid16), slog.String("error", "invalid command"), slog.String("cmd", cmd), slog.Int("status", http.StatusBadRequest))
		return
	}
	if !h.listAllowedCmd.MatchString(cmd) {
		h.writeJSONError(http.StatusBadRequest, "invalid command")
		h.log.Error("HTTP request error", slog.String("connection-uuid", uuid16), slog.String("error", "invalid command"), slog.String("cmd", cmd), slog.Int("status", http.StatusBadRequest))
		return
	}
	scriptPath := filepath.Join(h.cfg.ComDir, cmd+".lua")
	if _, err := os.Stat(scriptPath); err != nil {
		h.writeJSONError(http.StatusNotFound, "command not found")
		h.log.Error("HTTP request error", slog.String("connection-uuid", uuid16), slog.String("error", "command not found"), slog.String("cmd", cmd), slog.Int("status", http.StatusNotFound))
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

	L.DoString(`
		print = function() end
		io.write = function(...) end
		io.stdout = function() return nil end
		io.stderr = function() return nil end
		io.read = function(...) return nil end
	`)

	if err := L.DoFile(scriptPath); err != nil {
		h.writeJSONError(http.StatusInternalServerError, "lua error: "+err.Error())
		h.log.Error("Failed to execute lua script", slog.String("connection-uuid", uuid16), slog.String("error", err.Error()))
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
		h.log.Info("Command executed with error", slog.String("connection-uuid", uuid16), slog.String("cmd", cmd), slog.Any("result", out))
	case "ok":
		h.log.Info("Command executed successfully", slog.String("connection-uuid", uuid16), slog.String("cmd", cmd), slog.Any("result", out))
	default:
		h.log.Info("Command executed and returned an unknown status", slog.String("connection-uuid", uuid16), slog.String("cmd", cmd), slog.Any("result", out))
	}
	h.log.Info("Session completed", slog.String("connection-uuid", uuid16), slog.String("remote", h.r.RemoteAddr), slog.String("method", h.r.Method), slog.String("url", h.r.URL.String()))
}
