package sv1

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	lua "github.com/yuin/gopher-lua"
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
