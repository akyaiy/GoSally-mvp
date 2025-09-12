local session = require("internal.session")
local net = require("internal.net")
local log = require("internal.log")

local reqAddr
local logReq = true
local payload

log.debug(session.request.params)

if not (session.request.params and session.request.params.url) then
  session.response.error = {
    code = -32602,
    message = "no url or payload provided"
  }
  return
end



reqAddr = session.request.params.url
payload = session.request.params.payload

local resp = net.http.post_request(logReq, reqAddr, "application/json", payload)
if resp then
  session.response.result.answer = {
    status = resp.status,
    body = resp.body
  }
  return
end

session.response.error = {
  data = "error while requesting"
}