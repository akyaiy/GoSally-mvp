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

---@class HttpResponse
---@field status integer HTTP status code
---@field status_text string HTTP status text
---@field body string Response body
---@field content_length integer Content length
---@field headers table<string, string|string[]> Response headers

---@class Http
---@field Get fun(log: boolean, url: string): HttpResponse, string? Makes HTTP GET request
---@field Post fun(log: boolean, url: string, content_type: string, payload: string): HttpResponse, string? Makes HTTP POST request

---@class Net
---@field Http Http HTTP client methods

---@type Net
Net = {
    Http = {
        ---Makes HTTP GET request
        ---@param log boolean Whether to log the request
        ---@param url string URL to request
        ---@return HttpResponse response
        ---@return string? error
        Get = function(log, url) end,

        ---Makes HTTP POST request
        ---@param log boolean Whether to log the request
        ---@param url string URL to request
        ---@param content_type string Content-Type header
        ---@param payload string Request body
        ---@return HttpResponse response
        ---@return string? error
        Post = function(log, url, content_type, payload) end
    }
}