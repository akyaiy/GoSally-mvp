---@diagnostic disable: missing-fields, missing-return
---@alias AnyTable table<string, any>

---@type AnyTable
In = {
	Params = {},
}

---@type AnyTable
Out = {
	Result = {},
}

---@class Log
---@field Info fun(msg: string)
---@field Debug fun(msg: string)
---@field Error fun(msg: string)
---@field Warn fun(msg: string)
---@field Event fun(msg: string)
---@field EventError fun(msg: string)
---@field EventWarn fun(msg: string)

---@type Log
Log = {}