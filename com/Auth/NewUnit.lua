---@diagnostic disable: redefined-local
local db = require("internal.database-sqlite").connect("db/user-database.db", {log = true})
local log = require("internal.log")
local session = require("internal.session")
local crypt = require("internal.crypt.bcrypt")

if not session.request.params then
  session.response.error = {
    message = "no params provided"
  }
  return
end

local params = session.request.params

if not (params.username and params.email and params.password) then
  session.response.error = {
    message = "no username/email/password provided"
  }
  return
end

local hashPass = crypt.generate(params.password, crypt.DefaultCost)

local existing, err = db:query("SELECT 1 FROM users WHERE email = ? OR username = ? LIMIT 1", {
  params.email,
  params.username
})
if err ~= nil then
  session.response.error = {
    message = "Database check failed: "..tostring(err)
  }
  log.error("Email check failed: "..tostring(err))
  return
end

if existing and #existing > 0 then
  session.response.error = {
    code = -32604,
    message = "Unit already exists"
  }
  return
end

local ctx, err = db:exec(
  "INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
  {
    params.username,
    params.email,
    hashPass
  }
)
if err ~= nil then
  session.response.error = {
    code = -32605,
    message = "Insert failed: "..tostring(err)
  }
  log.error("Insert failed: "..tostring(err))
  return
end

local res, err = ctx:wait()
if err ~= nil then
  session.response.error = {
    code = -32606,
    message = "Insert confirmation failed: "..tostring(err)
  }
  log.error("Insert confirmation failed: "..tostring(err))
  return
end

session.response.result = {
  rows_affected = res,
  message = "Unit created successfully"
}

db:close()