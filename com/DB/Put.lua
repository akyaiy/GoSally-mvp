---@diagnostic disable: redefined-local
local db = require("internal.database-sqlite").connect("db/test.db", {log = true})
local log = require("internal.log")
local session = require("internal.session")

local ctx, err = db:exec([[
  CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
  )
]])
if err ~= nil then
  log.event_error("Failed to create table: "..tostring(err))
  return
end

_, err = ctx:wait()
if err ~= nil then
  log.event_error("Table creation failed: "..tostring(err))
  return
end

if not (session.request.params.name and session.request.params.email) then
  session.response.error = {
    code = -32602,
    message = "Name and email are required"
  }
  return
end

local existing, err = db:query("SELECT 1 FROM users WHERE email = ? LIMIT 1", {
  session.request.params.email
})
if err ~= nil then
  session.response.error = {
    code = -32603,
    message = "Database check failed: "..tostring(err)
  }
  log.error("Email check failed: "..tostring(err))
  return
end

if existing and #existing > 0 then
  session.response.error = {
    code = -32604,
    message = "Email already exists"
  }
  return
end

local ctx, err = db:exec(
  "INSERT INTO users (name, email) VALUES (?, ?)", 
  {
    session.request.params.name,
    session.request.params.email
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
  success = true,
  rows_affected = res,
  message = "User created successfully"
}

db:close()