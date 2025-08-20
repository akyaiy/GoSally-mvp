package lua

import (
	"sync"

	lua "github.com/yuin/gopher-lua"
)

type LuaPool struct {
	pool sync.Pool
}

func NewLuaPool() *LuaPool {
	return &LuaPool{
		pool: sync.Pool{
			New: func() any {
				L := lua.NewState()
				
				return L
			},
		},
	}
}

func (lp *LuaPool) Get() *lua.LState {
	return lp.pool.Get().(*lua.LState)
}

func (lp *LuaPool) Put(L *lua.LState) {
	L.Close()

	newL := lua.NewState()
	
	lp.pool.Put(newL)
}
