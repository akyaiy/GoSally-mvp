-- com/PutNewUnit.lua

---@diagnostic disable: redefined-local
local db = require("internal.database-sqlite").connect("db/user-database.db", {log = true})
local log = require("internal.log")
local session = require("internal.session")
local crypt = require("internal.crypt.bcrypt")

local params = session.request.params.get()
local token = session.request.headers.get("x-session-token")

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
  return error_response("no params provided")
end

if not (token and token == require("_config").token()) then
  return error_response("access denied")
end

if not (params.username and params.email and params.password) then
  return error_response("no username/email/password provided")
end

local hashPass = crypt.generate(params.password, crypt.DefaultCost)

local existing, err = db:query("SELECT 1 FROM users WHERE deleted = 0 AND (email = ? OR username = ? OR phone_number = ?) LIMIT 1", {
  params.email,
  params.username,
  params.phone_number
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