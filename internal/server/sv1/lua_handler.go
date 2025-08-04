package sv1

// TODO: make a lua state pool using sync.Pool

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/akyaiy/GoSally-mvp/internal/colors"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
	lua "github.com/yuin/gopher-lua"
	_ "modernc.org/sqlite"
)

type DBConnection struct {
	dbPath    string
	log       bool
	logger    *slog.Logger
	writeChan chan *dbWriteRequest
	closeChan chan struct{}
}

type dbWriteRequest struct {
	query string
	args  []interface{}
	resCh chan *dbWriteResult
}

type dbWriteResult struct {
	rowsAffected int64
	err          error
}

var dbMutexMap = make(map[string]*sync.RWMutex)
var dbGlobalMutex sync.Mutex

func getDBMutex(dbPath string) *sync.RWMutex {
	dbGlobalMutex.Lock()
	defer dbGlobalMutex.Unlock()

	if mtx, ok := dbMutexMap[dbPath]; ok {
		return mtx
	}

	mtx := &sync.RWMutex{}
	dbMutexMap[dbPath] = mtx
	return mtx
}

func loadDBMod(llog *slog.Logger) func(*lua.LState) int {
	llog.Debug("import module db-sqlite")
	return func(L *lua.LState) int {
		dbMod := L.NewTable()

		L.SetField(dbMod, "connect", L.NewFunction(func(L *lua.LState) int {
			dbPath := L.CheckString(1)

			logQueries := false
			if L.GetTop() >= 2 {
				opts := L.CheckTable(2)
				if val := opts.RawGetString("log"); val != lua.LNil {
					logQueries = lua.LVAsBool(val)
				}
			}

			conn := &DBConnection{
				dbPath:    dbPath,
				log:       logQueries,
				logger:    llog,
				writeChan: make(chan *dbWriteRequest, 100),
				closeChan: make(chan struct{}),
			}

			go conn.processWrites()

			ud := L.NewUserData()
			ud.Value = conn
			L.SetMetatable(ud, L.GetTypeMetatable("gosally_db"))

			L.Push(ud)
			return 1
		}))

		mt := L.NewTypeMetatable("gosally_db")
		L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"exec":  dbExec,
			"query": dbQuery,
			"close": dbClose,
		}))

		L.SetField(dbMod, "__gosally_internal", lua.LString("0"))
		L.Push(dbMod)
		return 1
	}
}

func (conn *DBConnection) processWrites() {
	for {
		select {
		case req := <-conn.writeChan:
			mtx := getDBMutex(conn.dbPath)
			mtx.Lock()

			db, err := sql.Open("sqlite", conn.dbPath+"?_busy_timeout=5000&_journal_mode=WAL&_sync=NORMAL&_cache_size=-10000")
			if err == nil {
				_, err = db.Exec("PRAGMA journal_mode=WAL;")
				if err == nil {
					res, execErr := db.Exec(req.query, req.args...)
					if execErr == nil {
						rows, _ := res.RowsAffected()
						req.resCh <- &dbWriteResult{rowsAffected: rows}
					} else {
						req.resCh <- &dbWriteResult{err: execErr}
					}
				}
				db.Close()
			}

			if err != nil {
				req.resCh <- &dbWriteResult{err: err}
			}

			mtx.Unlock()
		case <-conn.closeChan:
			return
		}
	}
}

func dbExec(L *lua.LState) int {
	ud := L.CheckUserData(1)
	conn, ok := ud.Value.(*DBConnection)
	if !ok {
		L.Push(lua.LNil)
		L.Push(lua.LString("invalid database connection"))
		return 2
	}

	query := L.CheckString(2)
	
	var args []any
	if L.GetTop() >= 3 {
		params := L.CheckTable(3)
		params.ForEach(func(k lua.LValue, v lua.LValue) {
			args = append(args, ConvertLuaTypesToGolang(v))
		})
	}

	if conn.log {
		conn.logger.Info("DB Exec",
			slog.String("query", query),
			slog.Any("params", args))
	}

	resCh := make(chan *dbWriteResult, 1)
	conn.writeChan <- &dbWriteRequest{
		query: query,
		args:  args,
		resCh: resCh,
	}

	ctx := L.NewTable()
	L.SetField(ctx, "done", lua.LBool(false))
	
	var result lua.LValue = lua.LNil
	var errorMsg lua.LValue = lua.LNil

	L.SetField(ctx, "wait", L.NewFunction(func(lL *lua.LState) int {
		res := <-resCh
		L.SetField(ctx, "done", lua.LBool(true))
		
		if res.err != nil {
			errorMsg = lua.LString(res.err.Error())
			result = lua.LNil
		} else {
			result = lua.LNumber(res.rowsAffected)
			errorMsg = lua.LNil
		}
		
		if res.err != nil {
			lL.Push(lua.LNil)
			lL.Push(lua.LString(res.err.Error()))
			return 2
		}
		lL.Push(lua.LNumber(res.rowsAffected))
		lL.Push(lua.LNil)
		return 2
	}))

	L.SetField(ctx, "check", L.NewFunction(func(lL *lua.LState) int {
		select {
		case res := <-resCh:
			lL.SetField(ctx, "done", lua.LBool(true))
			if res.err != nil {
				errorMsg = lua.LString(res.err.Error())
				result = lua.LNil
				lL.Push(lua.LNil)
				lL.Push(lua.LString(res.err.Error()))
				return 2
			} else {
				result = lua.LNumber(res.rowsAffected)
				errorMsg = lua.LNil
				lL.Push(lua.LNumber(res.rowsAffected))
				lL.Push(lua.LNil)
				return 2
			}
		default:
			lL.Push(result)
			lL.Push(errorMsg)
			return 2
		}
	}))

	L.Push(ctx)
	L.Push(lua.LNil)
	return 2
}

func dbQuery(L *lua.LState) int {
	ud := L.CheckUserData(1)
	conn, ok := ud.Value.(*DBConnection)
	if !ok {
		L.Push(lua.LNil)
		L.Push(lua.LString("invalid database connection"))
		return 2
	}

	query := L.CheckString(2)

	var args []any
	if L.GetTop() >= 3 {
		params := L.CheckTable(3)
		params.ForEach(func(k lua.LValue, v lua.LValue) {
			args = append(args, ConvertLuaTypesToGolang(v))
		})
	}

	if conn.log {
		conn.logger.Info("DB Query",
			slog.String("query", query),
			slog.Any("params", args))
	}

	mtx := getDBMutex(conn.dbPath)
	mtx.RLock()
	defer mtx.RUnlock()

	db, err := sql.Open("sqlite", conn.dbPath+"?_busy_timeout=5000&_journal_mode=WAL&_sync=NORMAL&_cache_size=-10000")
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("query failed: %v", err)))
		return 2
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("get columns failed: %v", err)))
		return 2
	}

	result := L.NewTable()
	colCount := len(columns)
	values := make([]any, colCount)
	valuePtrs := make([]any, colCount)

	for rows.Next() {
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(fmt.Sprintf("scan failed: %v", err)))
			return 2
		}

		rowTable := L.NewTable()
		for i, col := range columns {
			val := values[i]
			if val == nil {
				L.SetField(rowTable, col, lua.LNil)
			} else {
				L.SetField(rowTable, col, ConvertGolangTypesToLua(L, val))
			}
		}
		result.Append(rowTable)
	}

	if err := rows.Err(); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("rows iteration failed: %v", err)))
		return 2
	}

	L.Push(result)
	return 1
}

func dbClose(L *lua.LState) int {
	ud := L.CheckUserData(1)
	conn, ok := ud.Value.(*DBConnection)
	if !ok {
		L.Push(lua.LFalse)
		L.Push(lua.LString("invalid database connection"))
		return 2
	}

	close(conn.closeChan)
	L.Push(lua.LTrue)
	return 1
}

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

	seed := rand.Int()

	loadSessionMod := func(lL *lua.LState) int {
		llog.Debug("import module session", slog.String("script", path))
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

		lL.SetField(inTable, "address", lua.LString(r.RemoteAddr))
		lL.SetField(sessionMod, "request", inTable)
		lL.SetField(sessionMod, "response", outTable)

		lL.SetField(sessionMod, "id", lua.LString(sid))

		lL.SetField(sessionMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
		lL.Push(sessionMod)
		return 1
	}

	loadLogMod := func(lL *lua.LState) int {
		llog.Debug("import module log", slog.String("script", path))
		logMod := lL.NewTable()

		logFuncs := map[string]func(string, ...any){
			"info":  llog.Info,
			"debug": llog.Debug,
			"error": llog.Error,
			"warn":  llog.Warn,
		}

		for name, logFunc := range logFuncs {
			fun := logFunc
			lL.SetField(logMod, name, lL.NewFunction(func(lL *lua.LState) int {
				msg := lL.Get(1)
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
			lL.SetField(logMod, fn.field, lL.NewFunction(func(lL *lua.LState) int {
				msg := lL.Get(1)
				converted := ConvertLuaTypesToGolang(msg)
				if fn.color != nil {
					h.x.Log.Printf("%s: %s: %s", fn.color(), path, converted)
				} else {
					h.x.Log.Printf("%s: %s", path, converted)
				}
				return 0
			}))
		}

		lL.SetField(logMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
		lL.Push(logMod)
		return 1
	}

	loadNetMod := func(lL *lua.LState) int {
		llog.Debug("import module net", slog.String("script", path))
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
				llog.Info("HTTP GET request",
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
				llog.Info("HTTP POST request",
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

		lL.SetField(netMod, "__gosally_internal", lua.LString(fmt.Sprint(seed)))
		lL.Push(netMod)
		return 1
	}

	L.PreloadModule("internal.session", loadSessionMod)
	L.PreloadModule("internal.log", loadLogMod)
	L.PreloadModule("internal.net", loadNetMod)
	L.PreloadModule("internal.database-sqlite", loadDBMod(llog))

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
		return rpc.NewResponse(map[string]any{
			"responsible-node": h.cs.UUID32,
		}, req.ID)
	}

	tag := sessionTbl.RawGetString("__gosally_internal")
	if tag.Type() != lua.LTString || tag.String() != fmt.Sprint(seed) {
		llog.Debug("stock session module is not imported: wrong seed", slog.String("script", path))
		return rpc.NewResponse(map[string]any{
			"responsible-node": h.cs.UUID32,
		}, req.ID)
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
	payload := make(map[string]any)
	if tbl, ok := resultVal.(*lua.LTable); ok {
		tbl.ForEach(func(k, v lua.LValue) { payload[k.String()] = ConvertLuaTypesToGolang(v) })
	} else {
		payload["message"] = ConvertLuaTypesToGolang(resultVal)
	}
	payload["responsible-node"] = h.cs.UUID32
	return rpc.NewResponse(payload, req.ID)
}
