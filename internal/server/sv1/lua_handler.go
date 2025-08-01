package sv1

// TODO: make a lua state pool using sync.Pool

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
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

	seed := rand.Int()

	loadSessionMod := func(lL *lua.LState) int {
		h.x.SLog.Debug("import module session", slog.String("script", path))
		sessionMod := lL.NewTable()
		inTable := lL.NewTable()
		paramsTable := lL.NewTable()
		if fetchedParams, ok := req.Params.(map[string]any); ok {
			for k, v := range fetchedParams {
				lL.SetField(paramsTable, k, ConvertGolangTypesToLua(lL, v))
			}
		}
		lL.SetField(inTable, "params", paramsTable)

		outTable := lL.NewTable()
		resultTable := lL.NewTable()
		lL.SetField(outTable, "result", resultTable)

		lL.SetField(sessionMod, "request", inTable)
		lL.SetField(sessionMod, "response", outTable)

		lL.Push(sessionMod)
		lL.SetField(sessionMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
		return 1
	}

	loadLogMod := func(lL *lua.LState) int {
		h.x.SLog.Debug("import module log", slog.String("script", path))
		logMod := lL.NewTable()

		logFuncs := map[string]func(string, ...any){
			"info":  h.x.SLog.Info,
			"debug": h.x.SLog.Debug,
			"error": h.x.SLog.Error,
			"warn":  h.x.SLog.Warn,
		}

		for name, logFunc := range logFuncs {
			fun := logFunc
			lL.SetField(logMod, name, lL.NewFunction(func(lL *lua.LState) int {
				msg := lL.ToString(1)
				fun(fmt.Sprintf("the script says: %s", msg), slog.String("script", path))
				return 0
			}))
		}

		lL.SetField(logMod, "event", lL.NewFunction(func(lL *lua.LState) int {
			msg := lL.ToString(1)
			h.x.Log.Printf("%s: %s", path, msg)
			return 0
		}))

		lL.SetField(logMod, "event_error", lL.NewFunction(func(lL *lua.LState) int {
			msg := lL.ToString(1)
			h.x.Log.Printf("%s: %s: %s", colors.PrintError(), path, msg)
			return 0
		}))

		lL.SetField(logMod, "event_warn", lL.NewFunction(func(lL *lua.LState) int {
			msg := lL.ToString(1)
			h.x.Log.Printf("%s: %s: %s", colors.PrintWarn(), path, msg)
			return 0
		}))

		lL.Push(logMod)
		lL.SetField(logMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
		return 1
	}

	loadNetMod := func(lL *lua.LState) int {
		h.x.SLog.Debug("import module net", slog.String("script", path))
		netMod := lL.NewTable()
		netModhttp := lL.NewTable()

		lL.SetField(netModhttp, "get_request", lL.NewFunction(func(lL *lua.LState) int {
			logRequest := lL.ToBool(1)
			url := lL.ToString(2)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				lL.Push(lua.LNil)
				lL.Push(lua.LString(err.Error()))
				return 2
			}

			addInitiatorHeaders(sid, r, req.Header)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				lL.Push(lua.LNil)
				lL.Push(lua.LString(err.Error()))
				return 2
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				lL.Push(lua.LNil)
				lL.Push(lua.LString(err.Error()))
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

			result := lL.NewTable()
			lL.SetField(result, "status", lua.LNumber(resp.StatusCode))
			lL.SetField(result, "status_text", lua.LString(resp.Status))
			lL.SetField(result, "body", lua.LString(body))
			lL.SetField(result, "content_length", lua.LNumber(resp.ContentLength))

			headers := lL.NewTable()
			for k, v := range resp.Header {
				lL.SetField(headers, k, ConvertGolangTypesToLua(lL, v))
			}
			lL.SetField(result, "headers", headers)

			lL.Push(result)
			return 1
		}))

		lL.SetField(netModhttp, "post_request", lL.NewFunction(func(lL *lua.LState) int {
			logRequest := lL.ToBool(1)
			url := lL.ToString(2)
			contentType := lL.ToString(3)
			payload := lL.ToString(4)

			body := strings.NewReader(payload)

			req, err := http.NewRequest("POST", url, body)
			if err != nil {
				lL.Push(lua.LNil)
				lL.Push(lua.LString(err.Error()))
				return 2
			}

			req.Header.Set("Content-Type", contentType)

			addInitiatorHeaders(sid, r, req.Header)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				lL.Push(lua.LNil)
				lL.Push(lua.LString(err.Error()))
				return 2
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				lL.Push(lua.LNil)
				lL.Push(lua.LString(err.Error()))
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

			result := lL.NewTable()
			lL.SetField(result, "status", lua.LNumber(resp.StatusCode))
			lL.SetField(result, "status_text", lua.LString(resp.Status))
			lL.SetField(result, "body", lua.LString(respBody))
			lL.SetField(result, "content_length", lua.LNumber(resp.ContentLength))

			headers := lL.NewTable()
			for k, v := range resp.Header {
				lL.SetField(headers, k, ConvertGolangTypesToLua(lL, v))
			}
			lL.SetField(result, "headers", headers)

			lL.Push(result)
			return 1
		}))

		lL.SetField(netMod, "http", netModhttp)

		lL.Push(netMod)
		lL.SetField(netMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
		return 1
	}

	L.PreloadModule("session", loadSessionMod)
	L.PreloadModule("log", loadLogMod)
	L.PreloadModule("net", loadNetMod)

	prep := filepath.Join(*h.x.Config.Conf.Node.ComDir, "_prepare.lua")
	if _, err := os.Stat(prep); err == nil {
		if err := L.DoFile(prep); err != nil {
			h.x.SLog.Error("script error", slog.String("script", path), slog.String("error", err.Error()))
			return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
		}
	}
	if err := L.DoFile(path); err != nil {
		h.x.SLog.Error("script error", slog.String("script", path), slog.String("error", err.Error()))
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	pkg := L.GetGlobal("package")
	pkgTbl, ok := pkg.(*lua.LTable)
	if !ok {
		h.x.SLog.Error("script error", slog.String("script", path), slog.String("error", "package not found"))
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	loaded := pkgTbl.RawGetString("loaded")
	loadedTbl, ok := loaded.(*lua.LTable)
	if !ok {
		h.x.SLog.Error("script error", slog.String("script", path), slog.String("error", "package.loaded not found"))
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	sessionVal := loadedTbl.RawGetString("session")
	sessionTbl, ok := sessionVal.(*lua.LTable)
	if !ok {
		return rpc.NewResponse(map[string]any{
			"responsible-node": h.cs.UUID32,
		}, req.ID)
	}

	tag := sessionTbl.RawGetString("__gosally_internal")
	if tag.Type() != lua.LTString || tag.String() != fmt.Sprint(seed) {
		return rpc.NewResponse(map[string]any{
			"responsible-node": h.cs.UUID32,
		}, req.ID)
	}

	outVal := sessionTbl.RawGetString("response")
	outTbl, ok := outVal.(*lua.LTable)
	if !ok {
		h.x.SLog.Error("script error", slog.String("script", path), slog.String("error", "response is not a table"))
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	if errVal := outTbl.RawGetString("error"); errVal != lua.LNil {
		if errTbl, ok := errVal.(*lua.LTable); ok {
			code := rpc.ErrInternalError
			message := rpc.ErrInternalErrorS
			data := make(map[string]any)
			if c := errTbl.RawGetString("code"); c.Type() == lua.LTNumber {
				code = int(c.(lua.LNumber))
			}
			if msg := errTbl.RawGetString("message"); msg.Type() == lua.LTString {
				message = msg.String()
			}
			rawData := errTbl.RawGetString("data")

			if tbl, ok := rawData.(*lua.LTable); ok {
				tbl.ForEach(func(k, v lua.LValue) {data[k.String()] = ConvertLuaTypesToGolang(v)})
			} else {
				h.x.SLog.Error("the script terminated with an error", slog.String("code", strconv.Itoa(code)), slog.String("message", message))
				return rpc.NewError(code, message, rawData, req.ID)
			}
			h.x.SLog.Error("the script terminated with an error", slog.String("code", strconv.Itoa(code)), slog.String("message", message))
			return rpc.NewError(code, message, data, req.ID)
		}
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	resultVal := outTbl.RawGetString("result")
	payload := make(map[string]any)
	if tbl, ok := resultVal.(*lua.LTable); ok {
		tbl.ForEach(func(k, v lua.LValue) { payload[k.String()] = ConvertLuaTypesToGolang(v) })
	} else {
		payload["message"] = ConvertLuaTypesToGolang(resultVal)
	}
	payload["responsible-node"] = h.cs.UUID32
	return rpc.NewResponse(payload, req.ID)
}
