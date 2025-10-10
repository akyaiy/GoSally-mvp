package lua

import (
	"net/http"

	"github.com/akyaiy/GoSally-mvp/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/internal/engine/app"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
)

type LuaEngineDeps struct {
	HttpRequest    *http.Request
	JSONRPCRequest *rpc.RPCRequest
	SessionUUID string
	ScriptPath string
}

type LuaEngineContract interface {
	Handle(deps *LuaEngineDeps) *rpc.RPCResponse
	
}

type LuaEngine struct {
	x  *app.AppX
	cs *corestate.CoreState
}
