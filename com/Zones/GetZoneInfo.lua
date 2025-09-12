local session = require("internal.session")
local log = require("internal.log")
local jwt = require("internal.crypt.jwt")
local bc = require("internal.crypt.bcrypt")
local sha256 = require("internal.crypt.sha256")
local dbdriver = require("internal.database.sqlite")

local db_root = dbdriver.connect("db/root.db", {log = true})
local db_zone = nil

local function close_db()
  if db_root then
    db_root:close()
    db_root = nil
  end
  if db_zone then
    db_zone:close()
    db_zone = nil
  end
end

local token = session.request.headers.get("authorization")

if not token or type(token) ~= "string" then
  close_db()
  session.response.send_error(-32050, "Access denied")
end

local prefix = "Bearer "
if token:sub(1, #prefix) ~= prefix then
  close_db()
  session.response.send_error(-32052, "Invalid Authorization scheme")
end

local access_token = token:sub(#prefix + 1)

local err, data = jwt.decode(access_token, { secret = require("_config").token() })

if err or not data then
  close_db()
  session.response.send_error(-32053, "Cannod parse JWT", {err})
end

if data.master_id then
  
  
end

local params = session.request.params.get()

local function check_missing(arr, p)
  local is_missing = {}
  local ok = true
  for _, key in ipairs(arr) do
    if p[key] == nil then
      table.insert(is_missing, key)
      ok = false
    end
  end
  return ok, is_missing
end

local ok, mp = check_missing({"zone_name"}, params)
if not ok then
  close_db()
  session.response.send_error(-32602, "Missing params", mp)
end

close_db()