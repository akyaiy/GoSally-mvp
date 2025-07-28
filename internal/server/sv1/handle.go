package sv1

import (
	"net/http"

	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
)

func (h *HandlerV1) Handle(w http.ResponseWriter, r *http.Request, req rpc.RPCRequest) {
	w.Write([]byte("Sigmas"))
}

// func (h *HandlerV1) Handle(w http.ResponseWriter, r *http.Request) {
// 	var req PettiRequest
// 	// server, ok := s.servers[serversApiVer(payload.PettiVer)]
// 	// if !ok {
// 	// 	WriteRouterError(w, &RouterError{
// 	// 		Status:     "error",
// 	// 		StatusCode: http.StatusBadRequest,
// 	// 		Payload: map[string]any{
// 	// 			"Message": InvalidProtovolVersion,
// 	// 		},
// 	// 	})
// 	// 	s.log.Info("invalid request received", slog.String("issue", InvalidProtovolVersion), slog.String("requested-version", payload.PettiVer))
// 	// 	return
// 	// }
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		utils.WriteJSONError(w, http.StatusBadRequest, "невалидный JSON: "+err.Error())
// 		return
// 	}

// 	if req.PettiVer == "" {
// 		utils.WriteJSONError(w, http.StatusBadRequest, "отсутствует PettiVer")
// 		return
// 	}
// 	if req.PettiVer != h.GetVersion() {
// 		utils.WriteJSONError(w, http.StatusBadRequest, "неподдерживаемая версия PettiVer")
// 		return
// 	}
// 	if req.PackageType.Request == "" {
// 		utils.WriteJSONError(w, http.StatusBadRequest, "отсутствует PackageType.Request")
// 		return
// 	}
// 	if req.Payload == nil {
// 		utils.WriteJSONError(w, http.StatusBadRequest, "отсутствует Payload")
// 		return
// 	}
// 	cmdRaw, ok := req.Payload["Exec"].(string)
// 	if !ok || cmdRaw == "" {
// 		utils.WriteJSONError(w, http.StatusBadRequest, "Payload.Exec отсутствует или некорректен")
// 		return
// 	}
// 	cmd := cmdRaw

// 	if !h.allowedCmd.MatchString(string([]rune(cmd)[0])) || !h.listAllowedCmd.MatchString(cmd) {
// 		utils.WriteJSONError(w, http.StatusBadRequest, "команда запрещена")
// 		return
// 	}

// 	// ===== Проверка скрипта
// 	scriptPath := h.comMatch(h.GetVersion(), cmd)
// 	if scriptPath == "" {
// 		utils.WriteJSONError(w, http.StatusNotFound, "команда не найдена")
// 		return
// 	}
// 	fullPath := filepath.Join(h.cfg.ComDir, scriptPath)
// 	if _, err := os.Stat(fullPath); err != nil {
// 		utils.WriteJSONError(w, http.StatusNotFound, "файл команды не найден")
// 		return
// 	}

// 	// ===== Запуск Lua
// 	L := lua.NewState()
// 	defer L.Close()

// 	inTable := L.NewTable()
// 	paramsTable := L.NewTable()
// 	if params, ok := req.Payload["PassedParameters"].(map[string]interface{}); ok {
// 		for k, v := range params {
// 			L.SetField(paramsTable, k, utils.ConvertGolangTypesToLua(L, v))
// 		}
// 	}
// 	L.SetField(inTable, "Params", paramsTable)
// 	L.SetGlobal("In", inTable)

// 	resultTable := L.NewTable()
// 	outTable := L.NewTable()
// 	L.SetField(outTable, "Result", resultTable)
// 	L.SetGlobal("Out", outTable)

// 	prepareLua := filepath.Join(h.cfg.ComDir, "_prepare.lua")
// 	if _, err := os.Stat(prepareLua); err == nil {
// 		if err := L.DoFile(prepareLua); err != nil {
// 			utils.WriteJSONError(w, http.StatusInternalServerError, "lua _prepare ошибка: "+err.Error())
// 			return
// 		}
// 	}
// 	if err := L.DoFile(fullPath); err != nil {
// 		utils.WriteJSONError(w, http.StatusInternalServerError, "lua exec ошибка: "+err.Error())
// 		return
// 	}

// 	lv := L.GetGlobal("Out")
// 	tbl, ok := lv.(*lua.LTable)
// 	if !ok {
// 		utils.WriteJSONError(w, http.StatusInternalServerError, "'Out' не таблица")
// 		return
// 	}
// 	resultVal := tbl.RawGetString("Result")
// 	resultTbl, ok := resultVal.(*lua.LTable)
// 	if !ok {
// 		utils.WriteJSONError(w, http.StatusInternalServerError, "'Result' не таблица")
// 		return
// 	}

// 	out := make(map[string]any)
// 	resultTbl.ForEach(func(key lua.LValue, value lua.LValue) {
// 		out[key.String()] = utils.ConvertLuaTypesToGolang(value)
// 	})

// 	uuid32, _ := corestate.GetNodeUUID(filepath.Join(config.MetaDir, "uuid"))

// 	resp := PettiResponse{
// 		PettiVer:             req.PettiVer,
// 		ResponsibleAgentUUID: uuid32,
// 		PackageType: struct {
// 			AnswerOf string `json:"AnswerOf"`
// 		}{AnswerOf: req.PackageType.Request},
// 		Payload: map[string]any{
// 			"RequestedCommand": cmd,
// 			"Response":         out,
// 		},
// 	}

// 	// ===== Финальная проверка на сериализацию (валидность сборки)
// 	respData, err := json.Marshal(resp)
// 	if err != nil {
// 		utils.WriteJSONError(w, http.StatusInternalServerError, "внутренняя ошибка: пакет невалиден")
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	if _, err := w.Write(respData); err != nil {
// 		h.log.Error("Ошибка при отправке JSON", slog.String("err", err.Error()))
// 	}

// 	// ===== Логгирование статуса
// 	status, _ := out["status"].(string)
// 	switch status {
// 	case "ok":
// 		h.log.Info("Успешно", slog.String("cmd", cmd), slog.Any("out", out))
// 	case "error":
// 		h.log.Warn("Ошибка в команде", slog.String("cmd", cmd), slog.Any("out", out))
// 	default:
// 		h.log.Info("Неизвестный статус", slog.String("cmd", cmd), slog.Any("out", out))
// 	}
// }

/*
import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/akyaiy/GoSally-mvp/core/config"
	"github.com/akyaiy/GoSally-mvp/core/corestate"
	"github.com/akyaiy/GoSally-mvp/core/utils"
	"github.com/go-chi/chi/v5"
	lua "github.com/yuin/gopher-lua"
)

// HandlerV1 is the main handler for version 1 of the API.
// The function processes the HTTP request and runs Lua scripts,
// preparing the environment and subsequently transmitting the execution result
func (h *HandlerV1) Handle(w http.ResponseWriter, r *http.Request) {
	uuid16, err := utils.NewUUID(int(config.UUIDLength))
	if err != nil {
		h.log.Error("Failed to generate UUID",
			slog.String("error", err.Error()))

		if err := utils.WriteJSONError(w, http.StatusInternalServerError, "failed to generate UUID: "+err.Error()); err != nil {
			h.log.Error("Failed to write JSON", slog.String("err", err.Error()))
		}
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

	cmd := chi.URLParam(r, "cmd")
	if !h.allowedCmd.MatchString(string([]rune(cmd)[0])) || !h.listAllowedCmd.MatchString(cmd) {
		log.Error("HTTP request error",
			slog.String("error", "invalid command"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusBadRequest))

		if err := utils.WriteJSONError(w, http.StatusBadRequest, "invalid command"); err != nil {
			h.log.Error("Failed to write JSON", slog.String("err", err.Error()))
		}
		return
	}

	scriptPath := h.comMatch(chi.URLParam(r, "ver"), cmd)
	if scriptPath == "" {
		log.Error("HTTP request error",
			slog.String("error", "command not found"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusNotFound))

		if err := utils.WriteJSONError(w, http.StatusNotFound, "command not found"); err != nil {
			h.log.Error("Failed to write JSON", slog.String("err", err.Error()))
		}
		return
	}

	scriptPath = filepath.Join(h.cfg.ComDir, scriptPath)
	if _, err := os.Stat(scriptPath); err != nil {
		log.Error("HTTP request error",
			slog.String("error", "command not found"),
			slog.String("cmd", cmd),
			slog.Int("status", http.StatusNotFound))

		if err := utils.WriteJSONError(w, http.StatusNotFound, "command not found"); err != nil {
			h.log.Error("Failed to write JSON", slog.String("err", err.Error()))
		}
		return
	}

	L := lua.NewState()
	defer L.Close()

	paramsTable := L.NewTable()
	qt := r.URL.Query()
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

			if err := utils.WriteJSONError(w, http.StatusInternalServerError, "lua error: "+err.Error()); err != nil {
				h.log.Error("Failed to write JSON", slog.String("err", err.Error()))
			}
			return
		}
	} else {
		log.Warn("No environment preparation script found, skipping preparation")
	}

	if err := L.DoFile(scriptPath); err != nil {
		log.Error("Failed to execute lua script",
			slog.String("error", err.Error()))
		if err := utils.WriteJSONError(w, http.StatusInternalServerError, "lua error: "+err.Error()); err != nil {
			h.log.Error("Failed to write JSON", slog.String("err", err.Error()))
		}
		return
	}

	lv := L.GetGlobal("Out")
	tbl, ok := lv.(*lua.LTable)
	if !ok {
		log.Error("Lua global 'Out' is not a table")

		if err := utils.WriteJSONError(w, http.StatusInternalServerError, "'Out' is not a table"); err != nil {
			h.log.Error("Failed to write JSON", slog.String("err", err.Error()))
		}
		return
	}

	resultVal := tbl.RawGetString("Result")
	resultTbl, ok := resultVal.(*lua.LTable)
	if !ok {
		log.Error("Lua global 'Result' is not a table")

		if err := utils.WriteJSONError(w, http.StatusInternalServerError, "'Result' is not a table"); err != nil {
			h.log.Error("Failed to write JSON", slog.String("err", err.Error()))
		}
		return
	}

	out := make(map[string]any)
	resultTbl.ForEach(func(key lua.LValue, value lua.LValue) {
		out[key.String()] = utils.ConvertLuaTypesToGolang(value)
	})
	uuid32, _ := corestate.GetNodeUUID(filepath.Join(config.MetaDir, "uuid"))
	response := ResponseFormat{
		ResponsibleAgentUUID: uuid32,
		RequestedCommand:     cmd,
		Response:             out,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
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
*/
