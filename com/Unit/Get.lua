-- File com/Unit/Get.lua
--
-- Created at 2025-09-25 20:04
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

local ok, mp = common.CheckMissingElement({"by", "value"}, params)
if not ok then
  close_db()
  session.response.send_error(errors.MISSING_PARAMS.code, errors.MISSING_PARAMS.message, mp)
end

if not (params.by == "email" or params.by == "username" or params.by == "user_id") then
  close_db()
  session.response.send_error(errors.INVALID_BY_PARAM.code, errors.INVALID_BY_PARAM.message)
end

local unit, err = db:query_row(
  "SELECT user_id, username, email, created_at, updated_at, deleted_at, entry_status FROM units WHERE "..params.by.." = ? AND deleted_at IS NULL LIMIT 1",
  {
    params.value
  }
)

if err then
  close_db()
  log.error("DB query error: " .. tostring(err))
  session.response.send_error()
end

if not unit then
  close_db()
  session.response.send_error(errors.UNIT_NOT_FOUND.code, errors.UNIT_NOT_FOUND.message)
end

close_db()
session.response.send(unit)