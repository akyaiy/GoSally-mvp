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
		h.writeJSONError(http.StatusBadRequest, "invalid command")
		return
	}

	scriptPath := h.comMatch(chi.URLParam(h.r, "ver"), cmd)
	if scriptPath == "" {
		log.Error("HTTP request error",
			slog.String("error", "command not found"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusNotFound))
		h.writeJSONError(http.StatusNotFound, "command not found")
		return
	}

	scriptPath = filepath.Join(h.cfg.ComDir, scriptPath)
	if _, err := os.Stat(scriptPath); err != nil {
		log.Error("HTTP request error",
			slog.String("error", "command not found"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusNotFound))
		h.writeJSONError(http.StatusNotFound, "command not found")
		return
	}

	L := lua.NewState()
	defer L.Close()

	// Создаем таблицу Params
	// Создаем таблицу In с Params
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

	// Скрипт подготовки окружения
	prepareLuaEnv := filepath.Join(h.cfg.ComDir, "_prepare.lua")
	if _, err := os.Stat(prepareLuaEnv); err == nil {
		if err := L.DoFile(prepareLuaEnv); err != nil {
			log.Error("Failed to prepare lua environment",
				slog.String("error", err.Error()))
			h.writeJSONError(http.StatusInternalServerError, "lua error: "+err.Error())
			return
		}
	} else {
		log.Warn("No environment preparation script found, skipping preparation")
	}

	// Основной Lua скрипт
	if err := L.DoFile(scriptPath); err != nil {
		log.Error("Failed to execute lua script",
			slog.String("error", err.Error()))
		h.writeJSONError(http.StatusInternalServerError, "lua error: "+err.Error())
		return
	}

	// Получаем Out
	lv := L.GetGlobal("Out")
	tbl, ok := lv.(*lua.LTable)
	if !ok {
		log.Error("Lua global 'Out' is not a table")
		h.writeJSONError(http.StatusInternalServerError, "'Out' is not a table")
		return
	}

	// Получаем Result из Out
	resultVal := tbl.RawGetString("Result")
	resultTbl, ok := resultVal.(*lua.LTable)
	if !ok {
		log.Error("Lua global 'Result' is not a table")
		h.writeJSONError(http.StatusInternalServerError, "'Result' is not a table")
		return
	}

	// Перебираем таблицу Result
	out := make(map[string]interface{})
	resultTbl.ForEach(func(key lua.LValue, value lua.LValue) {
		out[key.String()] = convertTypes(value)
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

func convertTypes(value lua.LValue) any {
	switch value.Type() {
	case lua.LTString:
		return value.String()
	case lua.LTNumber:
		return float64(value.(lua.LNumber))
	case lua.LTBool:
		return bool(value.(lua.LBool))
	case lua.LTTable:
		result := make(map[string]interface{})
		if tbl, ok := value.(*lua.LTable); ok {
			tbl.ForEach(func(key lua.LValue, value lua.LValue) {
				result[key.String()] = convertTypes(value)
			})
		}
		return result
	case lua.LTNil:
		return nil
	default:
		return value.String()
	}
}
