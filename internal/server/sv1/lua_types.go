package sv1

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

func ConvertLuaTypesToGolang(value lua.LValue) any {
	switch value.Type() {
	case lua.LTString:
		return value.String()
	case lua.LTNumber:
		return float64(value.(lua.LNumber))
	case lua.LTBool:
		return bool(value.(lua.LBool))
	case lua.LTTable:
		tbl := value.(*lua.LTable)

		var arr []any
		isArray := true
		tbl.ForEach(func(key, val lua.LValue) {
			if key.Type() != lua.LTNumber {
				isArray = false
			}
			arr = append(arr, ConvertLuaTypesToGolang(val))
		})

		if isArray {
			return arr
		}

		result := make(map[string]any)
		tbl.ForEach(func(key, val lua.LValue) {
			result[key.String()] = ConvertLuaTypesToGolang(val)
		})
		return result

	case lua.LTNil:
		return nil
	default:
		return value.String()
	}
}

func ConvertGolangTypesToLua(L *lua.LState, val any) lua.LValue {
	switch v := val.(type) {

	case nil:
		return lua.LNil

	case string:
		return lua.LString(v)
	case bool:
		return lua.LBool(v)
	case int:
		return lua.LNumber(v)
	case int8:
		return lua.LNumber(v)
	case int16:
		return lua.LNumber(v)
	case int32:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case uint:
		return lua.LNumber(v)
	case uint8:
		return lua.LNumber(v)
	case uint16:
		return lua.LNumber(v)
	case uint32:
		return lua.LNumber(v)
	case uint64:
		return lua.LNumber(v)
	case float32:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)

	case []string:
		tbl := L.NewTable()
		for i, s := range v {
			tbl.RawSetInt(i+1, lua.LString(s))
		}
		return tbl
	case []int:
		tbl := L.NewTable()
		for i, n := range v {
			tbl.RawSetInt(i+1, lua.LNumber(n))
		}
		return tbl
	case []float64:
		tbl := L.NewTable()
		for i, f := range v {
			tbl.RawSetInt(i+1, lua.LNumber(f))
		}
		return tbl
	case []any:
		tbl := L.NewTable()
		for i, item := range v {
			tbl.RawSetInt(i+1, ConvertGolangTypesToLua(L, item))
		}
		return tbl

	case map[string]string:
		tbl := L.NewTable()
		for k, s := range v {
			tbl.RawSetString(k, lua.LString(s))
		}
		return tbl
	case map[string]int:
		tbl := L.NewTable()
		for k, n := range v {
			tbl.RawSetString(k, lua.LNumber(n))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for k, val := range v {
			tbl.RawSetString(k, ConvertGolangTypesToLua(L, val))
		}
		return tbl

	default:
		return lua.LString(fmt.Sprintf("%v", v))
	}
}