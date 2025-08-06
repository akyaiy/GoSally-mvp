package sv1

import (
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt"
	lua "github.com/yuin/gopher-lua"
)

func loadJWTMod(llog *slog.Logger, sid string) func(*lua.LState) int {
	return func(L *lua.LState) int {
		llog.Debug("import module jwt")
		jwtMod := L.NewTable()

		L.SetField(jwtMod, "encode", L.NewFunction(jwtEncode))
		L.SetField(jwtMod, "decode", L.NewFunction(jwtDecode))

		L.SetField(jwtMod, "__gosally_internal", lua.LString(sid))
		L.Push(jwtMod)
		return 1
	}
}

func jwtEncode(L *lua.LState) int {
	payloadTbl := L.CheckTable(1)
	secret := L.GetField(payloadTbl, "secret").String()
	payload := L.GetField(payloadTbl, "payload").(*lua.LTable)
	expiresIn := L.GetField(payloadTbl, "expires_in")
	expDuration := time.Hour

	if expiresIn.Type() == lua.LTNumber {
		floatVal := ConvertLuaTypesToGolang(expiresIn).(float64)
		expDuration = time.Duration(floatVal) * time.Second
	}

	claims := jwt.MapClaims{}
	payload.ForEach(func(key, value lua.LValue) {
		claims[key.String()] = ConvertLuaTypesToGolang(value)
	})
	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(expDuration).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LString(signedToken))
	return 1
}

func jwtDecode(L *lua.LState) int {
	tokenString := L.CheckString(1)
	optsTbl := L.OptTable(2, L.NewTable())
	secret := L.GetField(optsTbl, "secret").String()

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		L.Push(lua.LString("Invalid token: " + err.Error()))
		L.Push(lua.LNil)
		return 2
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		L.Push(lua.LString("Invalid claims"))
		L.Push(lua.LNil)
		return 2
	}

	luaTable := L.NewTable()
	for k, v := range claims {
		luaTable.RawSetString(k, ConvertGolangTypesToLua(L, v))
	}

	L.Push(lua.LNil)
	L.Push(luaTable)
	return 2
}
