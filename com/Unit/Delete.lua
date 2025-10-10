-- File com/Unit/Delete.lua
--
-- Created at 2025-05-10 19:18
--
-- Updated at - 

local log = require("internal.log")
local db = require("internal.database.sqlite").connect("db/unit.db", {log = true})
local session = require("internal.session")

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

local ok, mp = common.CheckMissingElement({"user_id"}, params)
if not ok then
  close_db()
  session.response.send_error(errors.MISSING_PARAMS.code, errors.MISSING_PARAMS.message, mp)
end

local existing, err = db:query([[
  SELECT 1
  FROM units
  WHERE user_id = ?
    AND entry_status != 'deleted'
    AND deleted_at IS NULL
  LIMIT 1
]], {
  params.user_id
})

if err ~= nil then
  log.error("Email check failed: "..tostring(err))
  close_db()
  session.response.send_error()
end

if existing and #existing == 0 then
  close_db()
  session.response.send_error(errors.UNIT_NOT_FOUND.code, errors.UNIT_NOT_FOUND.message)
end

local ctx, err = db:exec(
  [[
    UPDATE units
    SET entry_status = 'deleted',
        deleted_at = CURRENT_TIMESTAMP
    WHERE user_id = ? AND deleted_at is NULL
  ]],
  { params.user_id }
)

if err ~= nil then
  log.error("Soft delete failed: " .. tostring(err))
  close_db()
  session.response.send_error(errors.DB_DELETE_FAILED.code, errors.DB_DELETE_FAILED.message)
end

local res, err = ctx:wait()
if err ~= nil then
  log.error("Soft delete confirmation failed: " .. tostring(err))
  close_db()
  session.response.send_error(errors.DB_DELETE_FAILED.code, errors.DB_DELETE_FAILED.message)
end

close_db()
session.response.send({message = "Unit deleted successfully", unit_id = params.user_id})