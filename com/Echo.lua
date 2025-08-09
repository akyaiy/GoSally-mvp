local s = require("internal.session")

if not s.request.params.__fetched.data then
  s.response.error = {
    code = 123,
    message = "params.data is missing"
  }
  return
end

s.response.send(s.request.params.__fetched)