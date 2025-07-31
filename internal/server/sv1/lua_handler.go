package sv1

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/akyaiy/GoSally-mvp/internal/colors"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
	lua "github.com/yuin/gopher-lua"
)

func addInitiatorHeaders(sid string, req *http.Request, headers http.Header) {
	clientIP := req.RemoteAddr
	if forwardedFor := req.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP = forwardedFor
	}
	headers.Set("X-Initiator-IP", clientIP)
	headers.Set("X-Session-UUID", sid)
	headers.Set("X-Initiator-Host", req.Host)
	headers.Set("X-Initiator-User-Agent", req.UserAgent())
	headers.Set("X-Initiator-Referer", req.Referer())
}

// A small reminder: this code is only at the MVP stage,
// and some parts of the code may cause shock from the
// incompetence of the developer. But, in the end,
// this code is just an idea. If there is a desire to
// contribute to the development of the code,
// I will be only glad.
// TODO: make this huge function more harmonious by dividing responsibilities
func (h *HandlerV1) handleLUA(sid string, r *http.Request, req *rpc.RPCRequest, path string) *rpc.RPCResponse {
	L := lua.NewState()
	defer L.Close()

	inTable := L.NewTable()
	paramsTable := L.NewTable()
	if fetchedParams, ok := req.Params.(map[string]any); ok {
		for k, v := range fetchedParams {
			L.SetField(paramsTable, k, ConvertGolangTypesToLua(L, v))
		}
	}
	L.SetField(inTable, "Params", paramsTable)
	L.SetGlobal("In", inTable)

	outTable := L.NewTable()
	resultTable := L.NewTable()
	L.SetField(outTable, "Result", resultTable)
	L.SetGlobal("Out", outTable)

	logTable := L.NewTable()

	logFuncs := map[string]func(string, ...any){
		"Info":  h.x.SLog.Info,
		"Debug": h.x.SLog.Debug,
		"Error": h.x.SLog.Error,
		"Warn":  h.x.SLog.Warn,
	}

	for name, logFunc := range logFuncs {
		L.SetField(logTable, name, L.NewFunction(func(L *lua.LState) int {
			msg := L.ToString(1)
			logFunc(fmt.Sprintf("the script says: %s", msg), slog.String("script", path))
			return 0
		}))
	}

	L.SetField(logTable, "Event", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.Log.Printf("%s: %s", path, msg)
		return 0
	}))

	L.SetField(logTable, "EventError", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.Log.Printf("%s: %s: %s", colors.PrintError(), path, msg)
		return 0
	}))

	L.SetField(logTable, "EventWarn", L.NewFunction(func(L *lua.LState) int {
		msg := L.ToString(1)
		h.x.Log.Printf("%s: %s: %s", colors.PrintWarn(), path, msg)
		return 0
	}))

	L.SetGlobal("Log", logTable)

	net := L.NewTable()
	netHttp := L.NewTable()

	L.SetField(netHttp, "Get", L.NewFunction(func(L *lua.LState) int {
		logRequest := L.ToBool(1)
		url := L.ToString(2)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}

		addInitiatorHeaders(sid, r, req.Header)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}

		if logRequest {
			h.x.SLog.Info("HTTP GET request",
				slog.String("script", path),
				slog.String("url", url),
				slog.Int("status", resp.StatusCode),
				slog.String("status_text", resp.Status),
				slog.String("initiator_ip", req.Header.Get("X-Initiator-IP")),
			)
		}

		result := L.NewTable()
		L.SetField(result, "status", lua.LNumber(resp.StatusCode))
		L.SetField(result, "status_text", lua.LString(resp.Status))
		L.SetField(result, "body", lua.LString(body))
		L.SetField(result, "content_length", lua.LNumber(resp.ContentLength))

		headers := L.NewTable()
		for k, v := range resp.Header {
			L.SetField(headers, k, ConvertGolangTypesToLua(L, v))
		}
		L.SetField(result, "headers", headers)

		L.Push(result)
		return 1
	}))

	L.SetField(netHttp, "Post", L.NewFunction(func(L *lua.LState) int {
		logRequest := L.ToBool(1)
		url := L.ToString(2)
		contentType := L.ToString(3)
		payload := L.ToString(4)

		body := strings.NewReader(payload)

		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}

		req.Header.Set("Content-Type", contentType)

		addInitiatorHeaders(sid, r, req.Header)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}

		if logRequest {
			h.x.SLog.Info("HTTP POST request",
				slog.String("script", path),
				slog.String("url", url),
				slog.String("content_type", contentType),
				slog.Int("status", resp.StatusCode),
				slog.String("status_text", resp.Status),
				slog.String("initiator_ip", req.Header.Get("X-Initiator-IP")),
			)
		}

		result := L.NewTable()
		L.SetField(result, "status", lua.LNumber(resp.StatusCode))
		L.SetField(result, "status_text", lua.LString(resp.Status))
		L.SetField(result, "body", lua.LString(respBody))
		L.SetField(result, "content_length", lua.LNumber(resp.ContentLength))

		headers := L.NewTable()
		for k, v := range resp.Header {
			L.SetField(headers, k, ConvertGolangTypesToLua(L, v))
		}
		L.SetField(result, "headers", headers)

		L.Push(result)
		return 1
	}))

	L.SetField(net, "Http", netHttp)
	L.SetGlobal("Net", net)

	prep := filepath.Join(*h.x.Config.Conf.Node.ComDir, "_prepare.lua")
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
		out[key.String()] = ConvertLuaTypesToGolang(value)
	})

	out["responsible-node"] = h.cs.UUID32
	return rpc.NewResponse(out, req.ID)
}
