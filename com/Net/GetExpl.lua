local session = require("session")
local net = require("net")

local reqAddr
local logReq = true

if session.request.params and session.request.params.url then
  reqAddr = session.request.params.url
else
  session.response.error = {
    code = -32602,
    message = "no url provided"
  }
  return
end

local resp = net.http.get_request(logReq, reqAddr)
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

