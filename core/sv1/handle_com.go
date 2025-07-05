package sv1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/akyaiy/GoSally-mvp/core/utils"
	"github.com/go-chi/chi/v5"
	lua "github.com/yuin/gopher-lua"
)

// HandlerV1 is the main handler for version 1 of the API.
// The function processes the HTTP request and runs Lua scripts, 
// preparing the environment and subsequently transmitting the execution result
func (h *HandlerV1) Handle(w http.ResponseWriter, r *http.Request) {
	uuid16, err := utils.NewUUID()
	if err != nil {
		h.log.Error("Failed to generate UUID",
			slog.String("error", err.Error()))
		utils.WriteJSONError(h.w, http.StatusInternalServerError, "failed to generate UUID: "+err.Error())
		return
	}
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

	cmd := chi.URLParam(h.r, "cmd")
	if !h.allowedCmd.MatchString(string([]rune(cmd)[0])) || !h.listAllowedCmd.MatchString(cmd) {
		log.Error("HTTP request error",
			slog.String("error", "invalid command"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusBadRequest))
		utils.WriteJSONError(h.w, http.StatusBadRequest, "invalid command")
		return
	}

	scriptPath := h.comMatch(chi.URLParam(h.r, "ver"), cmd)
	if scriptPath == "" {
		log.Error("HTTP request error",
			slog.String("error", "command not found"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusNotFound))
		utils.WriteJSONError(h.w, http.StatusNotFound, "command not found")
		return
	}

	scriptPath = filepath.Join(h.cfg.ComDir, scriptPath)
	if _, err := os.Stat(scriptPath); err != nil {
		log.Error("HTTP request error",
			slog.String("error", "command not found"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusNotFound))
		utils.WriteJSONError(h.w, http.StatusNotFound, "command not found")
		return
	}

	L := lua.NewState()
	defer L.Close()

	paramsTable := L.NewTable()
	qt := h.r.URL.Query()
	for k, v := range qt {
		if len(v) > 0 {
			L.SetField(paramsTable, k, lua.LString(v[0]))
		}
	}
	inTable := L.NewTable()
	L.SetField(inTable, "Params", paramsTable)
	L.SetGlobal("In", inTable)

	// Создаем таблицу Out с Result
	resultTable := L.NewTable()
	outTable := L.NewTable()
	L.SetField(outTable, "Result", resultTable)
	L.SetGlobal("Out", outTable)

	prepareLuaEnv := filepath.Join(h.cfg.ComDir, "_prepare.lua")
	if _, err := os.Stat(prepareLuaEnv); err == nil {
		if err := L.DoFile(prepareLuaEnv); err != nil {
			log.Error("Failed to prepare lua environment",
				slog.String("error", err.Error()))
			utils.WriteJSONError(h.w, http.StatusInternalServerError, "lua error: "+err.Error())
			return
		}
	} else {
		log.Warn("No environment preparation script found, skipping preparation")
	}

	if err := L.DoFile(scriptPath); err != nil {
		log.Error("Failed to execute lua script",
			slog.String("error", err.Error()))
		utils.WriteJSONError(h.w, http.StatusInternalServerError, "lua error: "+err.Error())
		return
	}

	lv := L.GetGlobal("Out")
	tbl, ok := lv.(*lua.LTable)
	if !ok {
		log.Error("Lua global 'Out' is not a table")
		utils.WriteJSONError(h.w, http.StatusInternalServerError, "'Out' is not a table")
		return
	}

	resultVal := tbl.RawGetString("Result")
	resultTbl, ok := resultVal.(*lua.LTable)
	if !ok {
		log.Error("Lua global 'Result' is not a table")
		utils.WriteJSONError(h.w, http.StatusInternalServerError, "'Result' is not a table")
		return
	}

	out := make(map[string]interface{})
	resultTbl.ForEach(func(key lua.LValue, value lua.LValue) {
		out[key.String()] = utils.ConvertLuaTypesToGolang(value)
	})

	h.w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(h.w).Encode(out); err != nil {
		log.Error("Failed to encode JSON response",
			slog.String("error", err.Error()))
	}

	status, _ := out["status"].(string)
	switch status {
	case "error":
		log.Info("Command executed with error",
			slog.String("cmd", cmd),
			slog.Any("result", out))
	case "ok":
		log.Info("Command executed successfully",
			slog.String("cmd", cmd),
			slog.Any("result", out))
	default:
		log.Info("Command executed and returned an unknown status",
			slog.String("cmd", cmd),
			slog.Any("result", out))
	}

	log.Info("Session completed")
}
