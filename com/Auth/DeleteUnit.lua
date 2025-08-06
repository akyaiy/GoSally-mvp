-- com/DeleteUnit.lua

---@diagnostic disable: redefined-local
local db = require("internal.database.sqlite").connect("db/user-database.db", {log = true})
local log = require("internal.log")
local session = require("internal.session")
local crypt = require("internal.crypt.bcrypt")
local jwt = require("internal.crypt.jwt")

local params = session.request.params.get()
local token = session.request.headers.get("authorization")

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

if not token or type(token) ~= "string" then
  return error_response("Access denied")
end

local prefix = "Bearer "
if token:sub(1, #prefix) ~= prefix then
  return error_response("Invalid Authorization scheme")
end

local access_token = token:sub(#prefix + 1)

local err, data = jwt.decode(access_token, { secret = require("_config").token() })

if err or not data then
  session.response.error = {
    message = err
  }
  return
end

if data.session_uuid ~= session.id then
  return error_response("Access denied")
end

if not params then
  return error_response("no params provided")
end

if not (params.username and params.email and params.password) then
  return error_response("no username/email/password provided")
end

local existing, err = db:query(
  "SELECT password FROM users WHERE email = ? AND username = ? AND deleted = 0 LIMIT 1",
  {
    params.email,
    params.username
  }
)

if err ~= nil then
  log.error("Password fetch failed: " .. tostring(err))
  return error_response("Database query failed: " .. tostring(err))
end

if not existing or #existing == 0 then
  return error_response("Unit not found")
end

local hashed_password = existing[1].password

local ok = crypt.compare(hashed_password, params.password)
if not ok then
  log.warn("Wrong password attempt for: " .. params.username)
  return error_response("Invalid password")
end

local ctx, err = db:exec(
  [[
    UPDATE users
    SET deleted = 1,
        deleted_at = CURRENT_TIMESTAMP
    WHERE email = ? AND username = ? AND deleted = 0
  ]],
  { params.email, params.username }
)

if err ~= nil then
  log.error("Soft delete failed: " .. tostring(err))
  return error_response("Soft delete failed: " .. tostring(err))
end

local res, err = ctx:wait()
if err ~= nil then
  log.error("Soft delete confirmation failed: " .. tostring(err))
  return error_response("Soft delete confirmation failed: " .. tostring(err))
end

session.response.result = {
  rows_affected = res,
  message = "Unit soft-deleted successfully"
}

log.info("user " .. params.username .. " soft-deleted successfully")

close_db()
