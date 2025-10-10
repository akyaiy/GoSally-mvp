-- File com/Unit/Update.lua
--
-- Created at 2025-10-10
--

local log = require("internal.log")
local db = require("internal.database.sqlite").connect("db/unit.db", { log = true })
local session = require("internal.session")

local common = require("com/Unit/_common")
local errors = require("com/Unit/_errors")

local function close_db()
  if db then
    log.debug("Closing DB connection")
    db:close()
    db = nil
  end
end

local params = session.request.params.get()

local ok, mp = common.CheckMissingElement({"user_id", "fields"}, params)
if not ok then
  close_db()
  session.response.send_error(errors.MISSING_PARAMS.code, errors.MISSING_PARAMS.message, mp)
end

if type(params.fields) ~= "table" or next(params.fields) == nil then
  close_db()
  session.response.send_error(errors.INVALID_FIELD_TYPE.code, errors.INVALID_FIELD_TYPE.message)
end

local allowed = {
  username = true,
  email = true,
  password = true,
  entry_status = true
}

local exists = db:query_row(
  "SELECT 1 FROM units WHERE user_id = ? AND deleted_at IS NULL LIMIT 1",
  { params.user_id }
)

if not exists then
  close_db()
  session.response.send_error(errors.UNIT_NOT_FOUND.code, errors.UNIT_NOT_FOUND.message)
end

local set_clauses = {}
local values = {}

for k, v in pairs(params.fields) do
  if allowed[k] then
    if k == "password" then
      local crypt = require("internal.crypt.bcrypt")
      v = crypt.generate(v, crypt.DefaultCost)
    end
    table.insert(set_clauses, k .. " = ?")
    table.insert(values, v)
  else
    log.warn("Ignoring unsupported field: " .. k)
  end
end

if #set_clauses == 0 then
  close_db()
  session.response.send_error(errors.NO_VALID_FIELDS.code, errors.NO_VALID_FIELDS.message)
end

table.insert(set_clauses, "updated_at = CURRENT_TIMESTAMP")

local query = "UPDATE units SET " .. table.concat(set_clauses, ", ")
  .. " WHERE user_id = ? AND deleted_at IS NULL"

table.insert(values, params.user_id)

local ctx, err = db:exec(query, values)
if not ctx then
  close_db()
  if tostring(err):match("UNIQUE constraint failed") then
    session.response.send_error(errors.UNIQUE_CONSTRAINT.code, errors.UNIQUE_CONSTRAINT.message)
  else
    session.response.send_error()
  end
end

local _, err = ctx:wait()
if err ~= nil then
  close_db()
  if tostring(err):match("UNIQUE constraint failed") then
    session.response.send_error(errors.UNIQUE_CONSTRAINT.code, errors.UNIQUE_CONSTRAINT.message)
  else
    log.error("Insert confirmation failed: "..tostring(err))
  session.response.send_error()
  end
end

close_db()

session.response.send({
  message = "User updated successfully",
  fields = params.fields
})
