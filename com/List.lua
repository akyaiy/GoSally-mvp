-- com/List.lua

if In.Params and In.Params.about then
  Out.Result = {
    description = "Returns a list of available methods",
    params = {
      layer = "select which layer list to display"
    }
  }
  return
end

local function isValidCommand(name)
  return name:match("^[%w]+$") ~= nil
end

local function scanDirectory(basePath, targetPath)
  local res = {}
  local fullPath = basePath.."/"..targetPath
  local handle = io.popen('find "'..fullPath..'" -type f -name "*.lua" 2>/dev/null')

  if handle then
    for filePath in handle:lines() do
      local fileName = filePath:match("([^/]+)%.lua$")
      if fileName and isValidCommand(fileName) then
        local relPath = filePath:gsub("^"..basePath.."/", ""):gsub(".lua$", ""):gsub("/", ">")
        table.insert(res, relPath)
      end
    end
    handle:close()
  end

  return #res > 0 and res or nil
end

local basePath = "com"
local layer = In.Params and In.Params.layer and In.Params.layer:gsub(">", "/") or nil

Out.Result = {
  answer = layer and scanDirectory(basePath, layer) or scanDirectory(basePath, "")
}