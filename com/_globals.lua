--@diagnostic disable: missing-fields, missing-return

---@alias Any any
---@alias AnyTable table<string, Any>

--- Global session module interface
---@class SessionIn
---@field params AnyTable Request parameters

---@class SessionOut
---@field result Any|string? Result payload (table or primitive)
---@field error { code: integer, message: string, data: Any }? Optional error info

---@class SessionModule
---@field request SessionIn Input context (read-only)
---@field response SessionOut Output context (write results/errors)

--- Global log module interface
---@class LogModule
---@field info fun(msg: string) Log informational message
---@field debug fun(msg: string) Log debug message
---@field error fun(msg: string) Log error message
---@field warn fun(msg: string) Log warning message
---@field event fun(msg: string) Log event (generic)
---@field event_error fun(msg: string) Log event error
---@field event_warn fun(msg: string) Log event warning

--- Global net module interface
---@class HttpResponse
---@field status integer HTTP status code
---@field status_text string HTTP status text
---@field body string Response body
---@field content_length integer Content length
---@field headers AnyTable Map of headers

---@class HttpModule
---@field get fun(log: boolean, url: string): HttpResponse, string? Perform GET
---@field post fun(log: boolean, url: string, content_type: string, payload: string): HttpResponse, string? Perform POST

---@class NetModule
---@field http HttpModule HTTP client functions

--- Global variables declaration
---@global
---@type SessionModule
_G.session = session or {}

---@global
---@type LogModule
_G.log = log or {}

---@global
---@type NetModule
_G.net = net or {}