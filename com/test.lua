local sha256 = require("internal.crypt.sha256")
local log = require("internal.log")
local session = require("internal.session")

-- local secret = require("_config").token()

-- local token = jwt.encode({
--   secret = secret,
--   payload = { session_uuid = session.id },
--   expires_in = 3600
-- })

-- local err, data = jwt.decode(token, { secret = secret })

-- if not err then
--   session.response.result = {
--     token = token
--   }
--   return
-- end

-- session.response.error = {
--   message = "not sigma"
-- }
-- local array = session.request.params.get("array", "oops")
-- function s()
--   session.throw_error("dqdqwdqwdqiwhodiwqohdq", 10)
-- end
-- s()

-- session.response.__script_data.result = {
--   data = {
--     sewf = 1
--   },
--   2
-- }
session.response.set_error()
--session.response.send_error({1})
-- session.response.set()
-- session.response.__script_data.result = {
--   status = "ok"
-- }
session.response.set(1)
log.event("popi")