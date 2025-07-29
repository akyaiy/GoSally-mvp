package sv1

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/akyaiy/GoSally-mvp/internal/core/utils"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
	lua "github.com/yuin/gopher-lua"
)

func (h *HandlerV1) Handle(r *http.Request, req *rpc.RPCRequest) *rpc.RPCResponse {
	if req.Method == "" {
		h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrMethodNotFoundS), slog.String("requested-method", req.Method))
		return rpc.NewError(rpc.ErrMethodIsMissing, rpc.ErrMethodIsMissingS, req.ID)
	}

	method, err := h.resolveMethodPath(req.Method)
	if err != nil {
		if err.Error() == rpc.ErrInvalidMethodFormatS {
			h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrInvalidMethodFormatS), slog.String("requested-method", req.Method))
			return rpc.NewError(rpc.ErrInvalidMethodFormat, rpc.ErrInvalidMethodFormatS, req.ID)
		} else if err.Error() == rpc.ErrMethodNotFoundS {
			h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrMethodNotFoundS), slog.String("requested-method", req.Method))
			return rpc.NewError(rpc.ErrMethodNotFound, rpc.ErrMethodNotFoundS, req.ID)
		}
	}

	return h.HandleLUA(method, req)
}

func (h *HandlerV1) HandleLUA(path string, req *rpc.RPCRequest) *rpc.RPCResponse {
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
