package sv1

import (
	"fmt"
	"reflect"
	"strconv"

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

		maxIdx := 0
		isArray := true

		var isNumeric = false
		tbl.ForEach(func(key, _ lua.LValue) {
			var numKey lua.LValue
			var ok bool
			switch key.Type() {
			case lua.LTString:
				numKey, ok = key.(lua.LString)
				if !ok {
					isArray = false
					return
				}
			case lua.LTNumber:
				numKey, ok = key.(lua.LNumber)
				if !ok {
					isArray = false
					return
				}
				isNumeric = true
			}

			num, err := strconv.Atoi(numKey.String())
			if err != nil {
				isArray = false
				return
			}
			if num < 1 {
				isArray = false
				return
			}
			if num > maxIdx {
				maxIdx = num
			}
		})

		if isArray {
			arr := make([]any, maxIdx)
			if isNumeric {
				for i := 1; i <= maxIdx; i++ {
					arr[i-1] = ConvertLuaTypesToGolang(tbl.RawGetInt(i))
				}
			} else {
				for i := 1; i <= maxIdx; i++ {
					arr[i-1] = ConvertLuaTypesToGolang(tbl.RawGetString(strconv.Itoa(i)))
				}
			}
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
	if val == nil {
		return lua.LNil
	}

	rv := reflect.ValueOf(val)
	rt := rv.Type()

	switch rt.Kind() {
	case reflect.String:
		return lua.LString(rv.String())
	case reflect.Bool:
		return lua.LBool(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return lua.LNumber(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return lua.LNumber(rv.Float())

	case reflect.Slice, reflect.Array:
		tbl := L.NewTable()
		for i := 0; i < rv.Len(); i++ {
			tbl.RawSetInt(i+1, ConvertGolangTypesToLua(L, rv.Index(i).Interface()))
		}
		return tbl

	case reflect.Map:
		if rt.Key().Kind() == reflect.String {
			tbl := L.NewTable()
			for _, key := range rv.MapKeys() {
				val := rv.MapIndex(key)
				tbl.RawSetString(key.String(), ConvertGolangTypesToLua(L, val.Interface()))
			}
			return tbl
		}

	default:
		return lua.LString(fmt.Sprintf("%v", val))
	}
	return lua.LString(fmt.Sprintf("%v", val))
}
