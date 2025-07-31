local resp = Net.Http.Get(true, "https://google.com")
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

