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
	llog := h.x.SLog.With(slog.String("session-id", sid))
	llog.Debug("handling LUA")
	L := lua.NewState()
	defer L.Close()

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

		fetchedParamsTable := L.NewTable()
		if fetchedParams, ok := req.Params.(map[string]any); ok {
			for k, v := range fetchedParams {
				L.SetField(fetchedParamsTable, k, ConvertGolangTypesToLua(L, v))
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

			val := get(fetchedParamsTable, path)
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

		L.SetField(paramsTable, "__fetched", fetchedParamsTable)

		L.SetField(paramsTable, "get", paramsGetter)
		L.SetField(inTable, "params", paramsTable)

		outTable := L.NewTable()

		L.SetField(inTable, "address", lua.LString(r.RemoteAddr))
		L.SetField(sessionMod, "request", inTable)
		L.SetField(sessionMod, "response", outTable)

		L.SetField(sessionMod, "id", lua.LString(sid))

		L.SetField(sessionMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
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

		L.SetField(logMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
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

		L.SetField(netMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
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

		L.SetField(bcryptMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
		L.Push(bcryptMod)
		return 1
	}

	loadCryptbsha256Mod := func(L *lua.LState) int {
		llog.Debug("import module crypt.sha256", slog.String("script", path))
		sha265mod := L.NewTable()

		L.SetField(sha265mod, "sum", L.NewFunction(func(l *lua.LState) int {
			data := ConvertLuaTypesToGolang(L.Get(1))
			dataStr, ok := data.(string)
			if !ok {
				L.Push(lua.LNil)
				L.Push(lua.LString("error: data must be a string"))
				return 2
			}

			hash := sha256.Sum256([]byte(dataStr))

			L.Push(lua.LString(hex.EncodeToString(hash[:])))
			L.Push(lua.LNil)
			return 2
		}))

		L.SetField(sha265mod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
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
	if err := L.DoFile(path); err != nil {
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

	tag := sessionTbl.RawGetString("__gosally_internal")
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

	if errVal := outTbl.RawGetString("error"); errVal != lua.LNil {
		llog.Debug("catch error table", slog.String("script", path))
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
				tbl.ForEach(func(k, v lua.LValue) { data[k.String()] = ConvertLuaTypesToGolang(v) })
			} else {
				llog.Error("the script terminated with an error", slog.String("code", strconv.Itoa(code)), slog.String("message", message))
				return rpc.NewError(code, message, ConvertLuaTypesToGolang(rawData), req.ID)
			}
			llog.Error("the script terminated with an error", slog.String("code", strconv.Itoa(code)), slog.String("message", message))
			return rpc.NewError(code, message, data, req.ID)
		}
		return rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, req.ID)
	}

	resultVal := outTbl.RawGetString("result")
	if resultVal != lua.LNil {
    return rpc.NewResponse(ConvertLuaTypesToGolang(resultVal), req.ID)
	}
	return rpc.NewResponse(nil, req.ID)
}
