-- com/List.lua

local session = require("internal.session")

local params = session.request.params.get()

if params.about then
  session.response.result = {
    description = "Returns a list of available methods",
    params = {
      layer = "select which layer list to display"
    }
  }
  return
end

local function isValidName(name)
  return name:match("^[%w]+$") ~= nil
end

local function scanDirectory(basePath, targetPath)
  local res = {}
  local fullPath = basePath.."/"..targetPath
  local handle = io.popen('find "'..fullPath..'" -type f -name "*.lua" 2>/dev/null')

  if handle then
    for filePath in handle:lines() do
      local parts = {}
      for part in filePath:gsub(".lua$", ""):gmatch("[^/]+") do
        table.insert(parts, part)
      end

      local allValid = true
      for _, part in ipairs(parts) do
        if not isValidName(part) then
          allValid = false
          break
        end
      end

      if allValid then
        local relPath = filePath:gsub("^"..basePath.."/", ""):gsub(".lua$", ""):gsub("/", ">")
        table.insert(res, relPath)
      end
    end
    handle:close()
  end

  return #res > 0 and res or nil
end

local basePath = "com"
local layer = params.layer and params.layer:gsub(">", "/") or nil

session.response.result = {
  answer = layer and scanDirectory(basePath, layer) or scanDirectory(basePath, "")
}