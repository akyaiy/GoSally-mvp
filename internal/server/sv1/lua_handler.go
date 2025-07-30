package sv1

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/akyaiy/GoSally-mvp/internal/core/utils"
	"github.com/akyaiy/GoSally-mvp/internal/engine/logs"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
	lua "github.com/yuin/gopher-lua"
)

func (h *HandlerV1) handleLUA(path string, req *rpc.RPCRequest) *rpc.RPCResponse {
	L := lua.NewState()
	defer L.Close()

	inTable := L.NewTable()
	paramsTable := L.NewTable()
	if fetchedParams, ok := req.Params.(map[string]any); ok {
		for k, v := range fetchedParams {
			L.SetField(paramsTable, k, utils.ConvertGolangTypesToLua(L, v))
		}
	}
	L.SetField(inTable, "Params", paramsTable)
	L.SetGlobal("In", inTable)

	outTable := L.NewTable()
	resultTable := L.NewTable()
	L.SetField(outTable, "Result", resultTable)
	L.SetGlobal("Out", outTable)

	logTable := L.NewTable()

	L.SetField(logTable, "Info", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.SLog.Info(fmt.Sprintf("the script says: %s", msg), slog.String("script", path))
		return 0
	}))

	L.SetField(logTable, "Debug", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.SLog.Debug(fmt.Sprintf("the script says: %s", msg), slog.String("script", path))
		return 0
	}))

	L.SetField(logTable, "Error", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.SLog.Error(fmt.Sprintf("the script says: %s", msg), slog.String("script", path))
		return 0
	}))

	L.SetField(logTable, "Warn", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.SLog.Warn(fmt.Sprintf("the script says: %s", msg), slog.String("script", path))
		return 0
	}))

	L.SetField(logTable, "Event", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.Log.Printf("%s: %s", path, msg)
		return 0
	}))

	L.SetField(logTable, "EventError", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.Log.Printf("%s: %s: %s", logs.PrintError(), path, msg)
		return 0
	}))

	L.SetField(logTable, "EventWarn", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.Log.Printf("%s: %s: %s", logs.PrintWarn(), path, msg)
		return 0
	}))

	L.SetGlobal("Log", logTable)

	prep := filepath.Join(h.x.Config.Conf.ComDir, "_prepare.lua")
	if _, err := os.Stat(prep); err == nil {
		if err := L.DoFile(prep); err != nil {
			return rpc.NewError(rpc.ErrInternalError, err.Error(), req.ID)
		}
	}
	if err := L.DoFile(path); err != nil {
		return rpc.NewError(rpc.ErrInternalError, err.Error(), req.ID)
	}

	lv := L.GetGlobal("Out")
	outTbl, ok := lv.(*lua.LTable)
	if !ok {
		return rpc.NewError(rpc.ErrInternalError, "Out is not a table", req.ID)
	}

	// Check if Out.Error exists
	if errVal := outTbl.RawGetString("Error"); errVal != lua.LNil {
		if errTbl, ok := errVal.(*lua.LTable); ok {
			code := rpc.ErrInternalError
			message := "Internal script error"
			if c := errTbl.RawGetString("code"); c.Type() == lua.LTNumber {
				code = int(c.(lua.LNumber))
			}
			if msg := errTbl.RawGetString("message"); msg.Type() == lua.LTString {
				message = msg.String()
			}
			h.x.SLog.Error("the script terminated with an error", slog.String("code", strconv.Itoa(code)), slog.String("message", message))
			return rpc.NewError(code, message, req.ID)
		}
		return rpc.NewError(rpc.ErrInternalError, "Out.Error is not a table", req.ID)
	}

	// Otherwise, parse Out.Result
	resultVal := outTbl.RawGetString("Result")
	resultTbl, ok := resultVal.(*lua.LTable)
	if !ok {
		return rpc.NewError(rpc.ErrInternalError, "Out.Result is not a table", req.ID)
	}

	out := make(map[string]any)
	resultTbl.ForEach(func(key lua.LValue, value lua.LValue) {
		out[key.String()] = utils.ConvertLuaTypesToGolang(value)
	})

	out["responsible-node"] = h.cs.UUID32
	return rpc.NewResponse(out, req.ID)
}
