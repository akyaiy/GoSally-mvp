package utils

import lua "github.com/yuin/gopher-lua"

func ConvertLuaTypesToGolang(value lua.LValue) any {
	switch value.Type() {
	case lua.LTString:
		return value.String()
	case lua.LTNumber:
		return float64(value.(lua.LNumber))
	case lua.LTBool:
		return bool(value.(lua.LBool))
	case lua.LTTable:
		result := make(map[string]interface{})
		if tbl, ok := value.(*lua.LTable); ok {
			tbl.ForEach(func(key lua.LValue, value lua.LValue) {
				result[key.String()] = ConvertLuaTypesToGolang(value)
			})
		}
		return result
	case lua.LTNil:
		return nil
	default:
		return value.String()
	}
}
