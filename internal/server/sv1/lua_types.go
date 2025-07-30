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

		// Попробуем как массив
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
	case string:
		return lua.LString(v)
	case bool:
		return lua.LBool(v)
	case int:
		return lua.LNumber(float64(v))
	case int64:
		return lua.LNumber(float64(v))
	case float32:
		return lua.LNumber(float64(v))
	case float64:
		return lua.LNumber(v)
	case []any:
		tbl := L.NewTable()
		for i, item := range v {
			tbl.RawSetInt(i+1, ConvertGolangTypesToLua(L, item))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for key, value := range v {
			tbl.RawSetString(key, ConvertGolangTypesToLua(L, value))
		}
		return tbl
	case nil:
		return lua.LNil
	default:
		return lua.LString(fmt.Sprintf("%v", v))
	}
}
