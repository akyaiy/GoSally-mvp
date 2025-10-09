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
  session.response.send_error(-32602, "Missing params", mp)
end

local hashPass = crypt.generate(params.password, crypt.DefaultCost)
local unitID = string.sub(sha256.hash(session.__seed), 1, 16)

-- First db query: check if username or email already exists among active users
local existing, err = db:query([[
  SELECT 1
  FROM units
  WHERE (email = ? OR username = ?)
    AND entry_status != 'deleted'
    AND deleted_at IS NULL
  LIMIT 1
]], {
  params.email,
  params.username
})

if err ~= nil then
  log.error("Email check failed: "..tostring(err))
  close_db()
  session.response.send_error()
end

if existing and #existing > 0 then
  close_db()
  session.response.send_error(-32101, "Unit is already exists")
end

-- Second db query: insert new unit
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
  session.response.send_error("Failed to create unit")
end

local res, err = ctx:wait()
if err ~= nil then
  log.error("Insert confirmation failed: "..tostring(err))
  close_db()
  session.response.send_error("Failed to create unit")
end

close_db()
session.response.send({message = "Unit created successfully", unit_id = unitID})