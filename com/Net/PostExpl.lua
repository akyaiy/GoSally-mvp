local reqAddr
local logReq = true
local payload

if not In.Params and In.Params.url or not In.Params.payload then
  Out.Error = {
    code = -32602,
    message = "no url or payload provided"
  }
  return
end

reqAddr = In.Params.url
payload = In.Params.payload

local resp = Net.Http.Post(logReq, reqAddr, "application/json", payload)
if resp then
  Out.Result.answer = {
    status = resp.status,
    body = resp.body
  }
  return
end

Out.Result.answer = {
  status = resp.status
}