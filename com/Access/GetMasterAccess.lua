local session = require("internal.session")
local log = require("internal.log")
local jwt = require("internal.crypt.jwt")
local bc = require("internal.crypt.bcrypt")
local db = require("internal.database.sqlite").connect("db/root.db", {log = true})
local sha256 = require("internal.crypt.sha256")

log.info("Someone at "..session.request.address.." trying to get master access")

local function close_db()
  if db then
    db:close()
    db = nil
  end
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

local ok, mp = check_missing({"master_secret", "master_name", "my_key"}, params)
if not ok then
  close_db()
  session.response.send_error(-32602, "Missing params", mp)
end

if type(params.master_secret) ~= "string" then
  close_db()
  session.response.send_error(-32050, "Access denied")
end

if type(params.master_name) ~= "string" then
  close_db()
  session.response.send_error(-32050, "Access denied")
end

local master, err = db:query_row("SELECT * FROM master_units WHERE master_name = ?", {params.master_name})

if not master then
  log.event("DB query failed:", err)
  close_db()
  session.response.send_error(-32050, "Access denied")
end

local ok = bc.compare(master.master_secret, params.master_secret)
if not ok then
  log.warn("Login failed: wrong password")
  close_db()
  session.response.send_error(-32050, "Access denied")
end

local token = jwt.encode({
  secret = require("_config").token(),
  payload = { 
    session_uuid = session.id,
    master_id = master.id,
    key = sha256.sum(params.my_key)
  },
  expires_in = 3600
})

close_db()
session.response.send({
  token = token
})

-- G7HgOgl72o7t7u7r
