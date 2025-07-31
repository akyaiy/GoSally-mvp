local reqAddr
local logReq = true

if In.Params and In.Params.url then
  reqAddr = In.Params.url
else
  Out.Error = {
    code = -32602,
    message = "no url provided"
  }
  return
end

local resp = Net.Http.Get(logReq, reqAddr)
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

