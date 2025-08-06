-- com/GetAccess

---@diagnostic disable: redefined-local
local db = require("internal.database.sqlite").connect("db/user-database.db", {log = true})
local log = require("internal.log")
local session = require("internal.session")
local crypt = require("internal.crypt.bcrypt")
local jwt = require("internal.crypt.jwt")
local sha256 = require("internal.crypt.sha256")

local params = session.request.params.get()
local secret = require("_config").token()

local function close_db()
  if db then
    db:close()
    db = nil
  end
end

local function error_response(message, code, data)
  session.response.error = {
    code = code or nil,
    message = message,
    data = data or nil
  }
  close_db()
end

if not params then
  return error_response("No params provided")
end

if not (params.username and params.email and params.password) then
  return error_response("Missing username, email or password")
end

local unit, err = db:query(
  "SELECT id, username, email, password, created_at FROM users WHERE email = ? AND username = ? AND deleted = 0 LIMIT 1",
  {
    params.email,
    params.username
  }
)

if err then
  log.error("DB query error: " .. tostring(err))
  return error_response("Database query failed")
end

if not unit or #unit == 0 then
  return error_response("Unit not found")
end

unit = unit[1]

local ok = crypt.compare(unit.password, params.password)
if not ok then
  log.warn("Login failed: wrong password for " .. params.username)
  return error_response("Invalid password")
end

local token = jwt.encode({
  secret = secret,
  payload = { session_uuid = session.id,
    admin_user = params.username,
    key = sha256.sum(session.request.address .. session.id .. session.request.headers.get("user-agent", "noagent"))
  },
  expires_in = 3600
})

session.response.result = {
  access_token = token
}

close_db()
