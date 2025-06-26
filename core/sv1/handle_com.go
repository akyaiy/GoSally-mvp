package sv1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	lua "github.com/aarzilli/golua/lua"

	"github.com/go-chi/chi/v5"
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
	var scriptPath string
	if !h.allowedCmd.MatchString(string([]rune(cmd)[0])) {
		log.Error("HTTP request error",
			slog.String("error", "invalid command"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusBadRequest))
		h.writeJSONError(http.StatusBadRequest, "invalid command")
		return
	}
	if !h.listAllowedCmd.MatchString(cmd) {
		log.Error("HTTP request error",
			slog.String("error", "invalid command"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusBadRequest))
		h.writeJSONError(http.StatusBadRequest, "invalid command")
		return
	}
	if scriptPath = h.comMatch(chi.URLParam(h.r, "ver"), cmd); scriptPath == "" {
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
	L.OpenLibs()

	// Создаем таблицу Params
	L.NewTable()
	paramsTableIndex := L.GetTop() // Индекс таблицы в стеке

	// Заполняем таблицу из query параметров
	qt := h.r.URL.Query()
	for k, v := range qt {
		if len(v) > 0 {
			L.PushString(v[0])              // Значение
			L.SetField(paramsTableIndex, k) // paramsTable[k] = v[0]
		}
	}

	// Помещаем Params в глобальные переменные
	L.SetGlobal("Params")

	// Создаем пустую таблицу Result
	L.NewTable()
	L.SetGlobal("Result")

	// Загружаем и выполняем скрипт подготовки окружения, если есть
	prepareLuaEnv := filepath.Join(h.cfg.ComDir, "_prepare.lua")
	if _, err := os.Stat(prepareLuaEnv); err == nil {
		if err := L.DoFile(prepareLuaEnv); err != nil {
			log.Error("Failed to prepare lua environment",
				slog.String("error", err.Error()))
			h.writeJSONError(http.StatusInternalServerError, "lua error: "+err.Error())
			return
		}
	} else {
		log.Error("No environment preparation script found, skipping preparation",
			slog.String("error", err.Error()))
	}

	// Выполняем основной Lua скрипт
	if err := L.DoFile(scriptPath); err != nil {
		log.Error("Failed to execute lua script",
			slog.Group("lua-status",
				slog.String("error", err.Error()),
				slog.String("lua-version", lua.LUA_VERSION)))
		h.writeJSONError(http.StatusInternalServerError, "lua error: "+err.Error())
		return
	}

	// Получаем глобальную переменную Result (таблица)
	L.GetGlobal("Result")
	if L.IsTable(-1) {
		out := make(map[string]interface{})

		L.PushNil() // Первый ключ
		for {
			if L.Next(-2) == 0 {
				break
			}
			// На стеке: -1 = value, -2 = key
			key := L.ToString(-2)
			var val interface{}

			switch L.Type(-1) {
			case lua.LUA_TSTRING:
				val = L.ToString(-1)
			case lua.LUA_TNUMBER:
				val = L.ToNumber(-1)
			case lua.LUA_TBOOLEAN:
				val = L.ToBoolean(-1)
			default:
				// fallback
				val = L.ToString(-1)
			}
			out[key] = val
			L.Pop(1) // Удаляем value, key остаётся для следующего L.Next
		}
		L.Pop(1) // Удаляем таблицу Result со стека

		h.w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(h.w).Encode(out); err != nil {
			log.Error("Failed to encode JSON response",
				slog.String("error", err.Error()))
		}

		switch out["status"] {
		case "error":
			log.Info("Command executed with error",
				slog.String("cmd", cmd),
				slog.Any("result", out))
		case "ok":
			log.Info("Command executed successfully",
				slog.String("cmd", cmd), slog.Any("result", out))
		default:
			log.Info("Command executed and returned an unknown status",
				slog.String("cmd", cmd),
				slog.Any("result", out))
		}
	} else {
		L.Pop(1) // убираем не таблицу из стека
		log.Error("Lua global 'Result' is not a table")
		h.writeJSONError(http.StatusInternalServerError, "'Result' is not a table")
		return
	}

	log.Info("Session completed",
		slog.Group("lua-status",
			slog.String("lua-version", lua.LUA_VERSION)))
}
