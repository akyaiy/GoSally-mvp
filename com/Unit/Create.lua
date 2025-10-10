-- File com/Unit/Create.lua
--
-- Created at 2025-05-10 18:23
--
-- Updated at - 
-- Description:
--- Creates a record in the unit.db database without 
--- requiring additional permissions. Requires username, 
--- password (hashing occurs at the server level), and email fields.

local log = require("internal.log")
local db = require("internal.database.sqlite").connect("db/unit.db", {log = true})
local session = require("internal.session")
local crypt = require("internal.crypt.bcrypt")
local sha256 = require("internal.crypt.sha256")

local common = require("com/Unit/_common")
local errors = require("com/Unit/_errors")

-- Preparing for first db query
local function close_db()
  if db then
    log.debug("Closing DB connection")
    db:close()
    db = nil
  end
end

local params = session.request.params.get()

local ok, mp = common.CheckMissingElement({"username", "password", "email"}, params)
if not ok then
  close_db()
  session.response.send_error(errors.MISSING_PARAMS.code, errors.MISSING_PARAMS.message, mp)
end

local hashPass = crypt.generate(params.password, crypt.DefaultCost)
local unitID = string.sub(sha256.hash(session.__seed), 1, 16)

local ctx, err = db:exec(
  "INSERT INTO units (user_id, username, email, password) VALUES (?, ?, ?, ?)",
  {
    unitID,
    params.username,
    params.email,
    hashPass,
  }
)

if err ~= nil then
  log.error("Insert failed: "..tostring(err))
  close_db()
  session.response.send_error(errors.DB_INSERT_FAILED.code, errors.DB_INSERT_FAILED.message)
end

local _, err = ctx:wait()
if err ~= nil then
  close_db()
  if tostring(err):match("UNIQUE constraint failed") then
    session.response.send_error(errors.UNIT_EXISTS.code, errors.UNIT_EXISTS.message)
  else
    log.error("Insert confirmation failed: "..tostring(err))
  session.response.send_error()
  end
end

close_db()
session.response.send({unit_id = unitID})