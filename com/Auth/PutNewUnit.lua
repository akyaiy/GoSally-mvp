-- com/PutNewUnit.lua

---@diagnostic disable: redefined-local
local db = require("internal.database.sqlite").connect("db/user-database.db", {log = true})
local log = require("internal.log")
local session = require("internal.session")
local crypt = require("internal.crypt.bcrypt")
local jwt = require("internal.crypt.jwt")
local sha256 = require("internal.crypt.sha256")

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

if data.key ~= sha256.sum(session.request.address .. session.id .. session.request.headers.get("user-agent", "noagent")) then
  return error_response("Access denied")
end

if not params then
  return error_response("no params provided")
end

if not (params.username and params.email and params.password) then
  return error_response("no username/email/password provided")
end

local hashPass = crypt.generate(params.password, crypt.DefaultCost)

local existing, err = db:query("SELECT 1 FROM users WHERE deleted = 0 AND (email = ? OR username = ?) LIMIT 1", {
  params.email,
  params.username
})

if err ~= nil then
  log.error("Email check failed: "..tostring(err))
  return error_response("Database check failed: "..tostring(err))
end

if existing and #existing > 0 then
  return error_response("Unit already exists")
end

local ctx, err = db:exec(
  "INSERT INTO users (username, email, password, first_name, last_name, phone_number) VALUES (?, ?, ?, ?, ?, ?)",
  {
    params.username,
    params.email,
    hashPass,
    params.first_name or "",
    params.last_name or "",
    params.phone_number or ""
  }
)
if err ~= nil then
  log.error("Insert failed: "..tostring(err))
  return error_response("Insert failed: "..tostring(err))
end

local res, err = ctx:wait()
if err ~= nil then
  log.error("Insert confirmation failed: "..tostring(err))
  return error_response("Insert confirmation failed: "..tostring(err))
end

session.response.result = {
  rows_affected = res,
  message = "Unit created successfully"
}

close_db()