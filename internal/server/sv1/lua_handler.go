package sv1

// TODO: make a lua state pool using sync.Pool

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/akyaiy/GoSally-mvp/internal/colors"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
	lua "github.com/yuin/gopher-lua"
	_ "modernc.org/sqlite"
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
	var __exit = -1

	llog := h.x.SLog.With(slog.String("session-id", sid))
	llog.Debug("handling LUA")
	L := lua.NewState()
	defer L.Close()

	osMod := L.GetGlobal("os").(*lua.LTable)
	L.SetField(osMod, "exit", lua.LNil)

	ioMod := L.GetGlobal("io").(*lua.LTable)
	for _, k := range []string{"write", "output", "flush", "read", "input"} {
		ioMod.RawSetString(k, lua.LNil)
	}
	L.Env.RawSetString("print", lua.LNil)

	for _, name := range []string{"stdout", "stderr", "stdin"} {
		stream := ioMod.RawGetString(name)
		if t, ok := stream.(*lua.LUserData); ok {
			t.Metatable = lua.LNil
		}
	}

	seed := rand.Int()

	loadSessionMod := func(L *lua.LState) int {
		llog.Debug("import module session", slog.String("script", path))
		sessionMod := L.NewTable()
		inTable := L.NewTable()
		paramsTable := L.NewTable()
		headersTable := L.NewTable()

		fetchedHeadersTable := L.NewTable()
		for k, v := range r.Header {
			L.SetField(fetchedHeadersTable, k, ConvertGolangTypesToLua(L, v))
		}

		headersGetter := L.NewFunction(func(L *lua.LState) int {
			path := L.OptString(1, "")
			def := L.Get(2)

			get := func(path string) lua.LValue {
				if path == "" {
					return fetchedHeadersTable
				}
				fetched := r.Header.Get(path)
				if fetched == "" {
					return lua.LNil
				}
				return lua.LString(fetched)
			}
			val := get(path)
			if val == lua.LNil && def != lua.LNil {
				L.Push(def)
			} else {
				L.Push(val)
			}
			return 1
		})

		L.SetField(headersTable, "__fetched", fetchedHeadersTable)

		L.SetField(headersTable, "get", headersGetter)
		L.SetField(inTable, "headers", headersTable)

		fetchedParamsTable := L.NewTable()
		switch params := req.Params.(type) {
		case map[string]any:
			for k, v := range params {
				L.SetField(fetchedParamsTable, k, ConvertGolangTypesToLua(L, v))
			}
		case []any:
			for i, v := range params {
				fetchedParamsTable.RawSetInt(i+1, ConvertGolangTypesToLua(L, v))
			}
		}

		paramsGetter := L.NewFunction(func(L *lua.LState) int {
			path := L.OptString(1, "")
			def := L.Get(2)

			get := func(tbl *lua.LTable, path string) lua.LValue {
				if path == "" {
					return tbl
				}
				current := tbl
				parts := strings.Split(path, ".")
				size := len(parts)
				for index, key := range parts {
					val := current.RawGetString(key)
					if tblVal, ok := val.(*lua.LTable); ok {
						current = tblVal
					} else {
						if index == size-1 {
							return val
						}
						return lua.LNil
					}
				}
				return current
			}

			paramsTbl := L.GetField(paramsTable, "__fetched") //
			val := get(paramsTbl.(*lua.LTable), path)         //
			if val == lua.LNil && def != lua.LNil {
				L.Push(def)
			} else {
				L.Push(val)
			}
			return 1
		})
		L.SetField(paramsTable, "__fetched", fetchedParamsTable)

		L.SetField(paramsTable, "get", paramsGetter)
		L.SetField(inTable, "params", paramsTable)

		outTable := L.NewTable()
		scriptDataTable := L.NewTable()
		L.SetField(outTable, "__script_data", scriptDataTable)

		L.SetField(inTable, "address", lua.LString(r.RemoteAddr))

		L.SetField(sessionMod, "throw_error", L.NewFunction(func(L *lua.LState) int {
			arg := L.Get(1)
			var msg string
			switch arg.Type() {
			case lua.LTString:
				msg = arg.String()
			case lua.LTNumber:
				msg = strconv.FormatFloat(float64(arg.(lua.LNumber)), 'f', -1, 64)
			default:
				L.ArgError(1, "expected string or number")
				return 0
			}

			L.RaiseError("%s", msg)
			return 0
		}))

		resTable := L.NewTable()
		L.SetField(scriptDataTable, "result", resTable)
		L.SetField(outTable, "send", L.NewFunction(func(L *lua.LState) int {
			res := L.Get(1)
			if res == lua.LNil {
				__exit = 0
				L.RaiseError("__successfull")
				return 0
			}

			resFTable := scriptDataTable.RawGetString("result")
			if resPTable, ok := res.(*lua.LTable); ok {
				resPTable.ForEach(func(key, value lua.LValue) {
					L.SetField(resFTable, key.String(), value)
				})
			} else {
				L.SetField(scriptDataTable, "result", res)
			}

			__exit = 0
			L.RaiseError("__successfull")
			return 0
		}))

		L.SetField(outTable, "set", L.NewFunction(func(L *lua.LState) int {
			res := L.Get(1)
			if res == lua.LNil {
				return 0
			}

			resFTable := scriptDataTable.RawGetString("result")
			if resPTable, ok := res.(*lua.LTable); ok {
				resPTable.ForEach(func(key, value lua.LValue) {
					L.SetField(resFTable, key.String(), value)
				})
			} else {
				L.SetField(scriptDataTable, "result", res)
			}
			return 0
		}))

		errTable := L.NewTable()
		L.SetField(scriptDataTable, "error", errTable)
		L.SetField(outTable, "send_error", L.NewFunction(func(L *lua.LState) int {
			var params [3]lua.LValue
			for i := range 3 {
				params[i] = L.Get(i + 1)
			}
			if errTable, ok := scriptDataTable.RawGetString("error").(*lua.LTable); ok {
				for _, v := range params {
					switch v.Type() {
					case lua.LTNumber:
						if n, ok := v.(lua.LNumber); ok {
							L.SetField(errTable, "code", n)
						}
					case lua.LTString:
						if s, ok := v.(lua.LString); ok {
							L.SetField(errTable, "message", s)
						}
					case lua.LTTable:
						if tbl, ok := v.(*lua.LTable); ok {
							L.SetField(errTable, "data", tbl)
						}
					}
				}
			}

			__exit = 1
			L.RaiseError("__unsuccessfull")
			return 0
		}))

		L.SetField(outTable, "set_error", L.NewFunction(func(L *lua.LState) int {
			var params [3]lua.LValue
			for i := range 3 {
				params[i] = L.Get(i + 1)
			}
			if errTable, ok := scriptDataTable.RawGetString("error").(*lua.LTable); ok {
				for _, v := range params {
					switch v.Type() {
					case lua.LTNumber:
						if n, ok := v.(lua.LNumber); ok {
							L.SetField(errTable, "code", n)
						}
					case lua.LTString:
						if s, ok := v.(lua.LString); ok {
							L.SetField(errTable, "message", s)
						}
					case lua.LTTable:
						if tbl, ok := v.(*lua.LTable); ok {
							L.SetField(errTable, "data", tbl)
						}
					}
				}
			}
			return 0
		}))

		L.SetField(sessionMod, "request", inTable)
		L.SetField(sessionMod, "response", outTable)

		L.SetField(sessionMod, "id", lua.LString(sid))

		L.SetField(sessionMod, "__seed", lua.LString(fmt.Sprint(seed)))
		L.Push(sessionMod)
		return 1
	}

	loadLogMod := func(L *lua.LState) int {
		llog.Debug("import module log", slog.String("script", path))
		logMod := L.NewTable()

		logFuncs := map[string]func(string, ...any){
			"info":  llog.Info,
			"debug": llog.Debug,
			"error": llog.Error,
			"warn":  llog.Warn,
		}

		for name, logFunc := range logFuncs {
			fun := logFunc
			L.SetField(logMod, name, L.NewFunction(func(L *lua.LState) int {
				msg := L.Get(1)
				converted := ConvertLuaTypesToGolang(msg)
				fun(fmt.Sprintf("the script says: %s", converted), slog.String("script", path))
				return 0
			}))
		}

		for _, fn := range []struct {
			field string
			pfunc func(string, ...any)
			color func() string
		}{
			{"event", h.x.Log.Printf, nil},
			{"event_error", h.x.Log.Printf, colors.PrintError},
			{"event_warn", h.x.Log.Printf, colors.PrintWarn},
		} {
			L.SetField(logMod, fn.field, L.NewFunction(func(L *lua.LState) int {
				msg := L.Get(1)
				converted := ConvertLuaTypesToGolang(msg)
				if fn.color != nil {
					h.x.Log.Printf("%s: %s: %s", fn.color(), path, converted)
				} else {
					h.x.Log.Printf("%s: %s", path, converted)
				}
				return 0
			}))
		}

		L.SetField(logMod, "__seed", lua.LString(fmt.Sprint(seed)))
		L.Push(logMod)
		return 1
	}

	loadNetMod := func(L *lua.LState) int {
		llog.Debug("import module net", slog.String("script", path))
		netMod := L.NewTable()
		netModhttp := L.NewTable()

		L.SetField(netModhttp, "get_request", L.NewFunction(func(L *lua.LState) int {
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
				llog.Info("HTTP GET request",
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

		L.SetField(netModhttp, "post_request", L.NewFunction(func(L *lua.LState) int {
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
				llog.Info("HTTP POST request",
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

		L.SetField(netMod, "http", netModhttp)

		L.SetField(netMod, "__seed", lua.LString(fmt.Sprint(seed)))
		L.Push(netMod)
		return 1
	}

	loadCryptbcryptMod := func(L *lua.LState) int {
		llog.Debug("import module crypt.bcrypt", slog.String("script", path))
		bcryptMod := L.NewTable()

		L.SetField(bcryptMod, "MinCost", lua.LNumber(bcrypt.MinCost))
		L.SetField(bcryptMod, "MaxCost", lua.LNumber(bcrypt.MaxCost))
		L.SetField(bcryptMod, "DefaultCost", lua.LNumber(bcrypt.DefaultCost))

		L.SetField(bcryptMod, "generate", L.NewFunction(func(l *lua.LState) int {
			password := ConvertLuaTypesToGolang(L.Get(1))
			passwordStr, ok := password.(string)
			if !ok {
				L.Push(lua.LNil)
				L.Push(lua.LString("error: password must be a string"))
				return 2
			}

			cost := ConvertLuaTypesToGolang(L.Get(2))
			costInt := bcrypt.DefaultCost
			switch v := cost.(type) {
			case int:
				costInt = v
			case float64:
				costInt = int(v)
			case nil:
				// ok, use DefaultCost
			default:
				L.Push(lua.LNil)
				L.Push(lua.LString("error: cost must be an integer"))
				return 2
			}

			hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwordStr), costInt)
			if err != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString("error: " + err.Error()))
				return 2
			}

			L.Push(lua.LString(string(hashBytes)))
			L.Push(lua.LNil)
			return 2
		}))

		L.SetField(bcryptMod, "compare", L.NewFunction(func(l *lua.LState) int {
			hash := ConvertLuaTypesToGolang(L.Get(1))
			hashStr, ok := hash.(string)
			if !ok {
				L.Push(lua.LString("error: hash must be a string"))
				return 1
			}
			password := ConvertLuaTypesToGolang(L.Get(2))
			passwordStr, ok := password.(string)
			if !ok {
				L.Push(lua.LString("error: password must be a string"))
				return 1
			}

			err := bcrypt.CompareHashAndPassword([]byte(hashStr), []byte(passwordStr))
			if err != nil {
				L.Push(lua.LFalse)
				return 1
			}
			L.Push(lua.LTrue)
			return 1
		}))

		L.SetField(bcryptMod, "__seed", lua.LString(fmt.Sprint(seed)))
		L.Push(bcryptMod)
		return 1
	}

	loadCryptbsha256Mod := func(L *lua.LState) int {
		llog.Debug("import module crypt.sha256", slog.String("script", path))
		sha265mod := L.NewTable()

		L.SetField(sha265mod, "sum", L.NewFunction(func(l *lua.LState) int {
			data := ConvertLuaTypesToGolang(L.Get(1))
			var dataStr = fmt.Sprint(data)

			hash := sha256.Sum256([]byte(dataStr))

			L.Push(lua.LString(hex.EncodeToString(hash[:])))
			L.Push(lua.LNil)
			return 2
		}))

		L.SetField(sha265mod, "__seed", lua.LString(fmt.Sprint(seed)))
		L.Push(sha265mod)
		return 1
	}

	L.PreloadModule("internal.session", loadSessionMod)
	L.PreloadModule("internal.log", loadLogMod)
	L.PreloadModule("internal.net", loadNetMod)
	L.PreloadModule("internal.database.sqlite", loadDBMod(llog, fmt.Sprint(seed)))
	L.PreloadModule("internal.crypt.bcrypt", loadCryptbcryptMod)
	L.PreloadModule("internal.crypt.sha256", loadCryptbsha256Mod)
	L.PreloadModule("internal.crypt.jwt", loadJWTMod(llog, fmt.Sprint(seed)))

	llog.Debug("preparing environment")
	prep := filepath.Join(*h.x.Config.Conf.Node.ComDir, "_prepare.lua")
	if _, err := os.Stat(prep); err == nil {
		if err := L.DoFile(prep); err != nil {
			llog.Error("script error", slog.String("script", path), slog.String("error", err.Error()))
			return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
		}
	}
	llog.Debug("executing script", slog.String("script", path))
	err := L.DoFile(path)
	if err != nil && __exit != 0 && __exit != 1 {
		llog.Error("script error", slog.String("script", path), slog.String("error", err.Error()))
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	pkg := L.GetGlobal("package")
	pkgTbl, ok := pkg.(*lua.LTable)
	if !ok {
		llog.Error("script error", slog.String("script", path), slog.String("error", "package not found"))
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	loaded := pkgTbl.RawGetString("loaded")
	loadedTbl, ok := loaded.(*lua.LTable)
	if !ok {
		llog.Error("script error", slog.String("script", path), slog.String("error", "package.loaded not found"))
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	sessionVal := loadedTbl.RawGetString("internal.session")
	sessionTbl, ok := sessionVal.(*lua.LTable)
	if !ok {
		return rpc.NewResponse(nil, req.ID)
	}

	tag := sessionTbl.RawGetString("__seed")
	if tag.Type() != lua.LTString || tag.String() != fmt.Sprint(seed) {
		llog.Debug("stock session module is not imported: wrong seed", slog.String("script", path))
		return rpc.NewResponse(nil, req.ID)
	}

	outVal := sessionTbl.RawGetString("response")
	outTbl, ok := outVal.(*lua.LTable)
	if !ok {
		llog.Error("script error", slog.String("script", path), slog.String("error", "response is not a table"))
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	if scriptDataTable, ok := outTbl.RawGetString("__script_data").(*lua.LTable); ok {
		switch __exit {
		case 1:
			if errTbl, ok := scriptDataTable.RawGetString("error").(*lua.LTable); ok {
				llog.Debug("catch error table", slog.String("script", path))
				code := rpc.ErrInternalError
				message := rpc.ErrInternalErrorS
				if c := errTbl.RawGetString("code"); c.Type() == lua.LTNumber {
					code = int(c.(lua.LNumber))
				}
				if msg := errTbl.RawGetString("message"); msg.Type() == lua.LTString {
					message = msg.String()
				}
				data := ConvertLuaTypesToGolang(errTbl.RawGetString("data"))
				llog.Error("the script terminated with an error", slog.Int("code", code), slog.String("message", message), slog.Any("data", data))
				return rpc.NewError(code, message, data, req.ID)
			}
			return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
		case 0:
			resVal := ConvertLuaTypesToGolang(scriptDataTable.RawGetString("result"))
			return rpc.NewResponse(resVal, req.ID)
		}
	}
	return rpc.NewResponse(nil, req.ID)
}
