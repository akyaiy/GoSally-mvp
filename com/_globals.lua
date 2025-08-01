--@diagnostic disable: missing-fields, missing-return

---@alias Any any
---@alias AnyTable table<string, Any>

--- Global session module interface
---@class SessionModule
---@field request AnyTable     Input context (read-only)
---@field request.params AnyTable     Request parameters
---@field response AnyTable    Output context (write results/errors)
---@field response.result Any|string?  Result payload (table or primitive)
---@field response.error { code: integer, message: string, data: any }?  Optional error info

--- Global log module interface
---@class LogModule
---@field info fun(msg: string)                    Log informational message
---@field debug fun(msg: string)                   Log debug message
---@field error fun(msg: string)                   Log error message
---@field warn fun(msg: string)                    Log warning message
---@field event fun(msg: string)                   Log event (generic)
---@field event_error fun(msg: string)             Log event error
---@field event_warn fun(msg: string)              Log event warning

--- Global net module interface
---@class HttpResponse
---@field status integer        HTTP status code
---@field status_text string    HTTP status text
---@field body string           Response body
---@field content_length integer Content length
---@field headers AnyTable      Map of headers

---@class HttpModule
---@field get fun(log: boolean, url: string): HttpResponse, string?   Perform GET
---@field post fun(log: boolean, url: string, content_type: string, payload: string): HttpResponse, string?  Perform POST

---@class NetModule
---@field http HttpModule       HTTP client functions

--- Exposed globals
---@type SessionModule
session = session or {}

---@type LogModule
log = log or {}

---@type NetModule
net = net or {}