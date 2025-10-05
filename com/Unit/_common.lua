-- File com/Unit/_common.lua
--
-- Created at 2025-05-10 18:23
--
-- Updated at - 
-- Description:
--- Common functions for Unit module

local common = {}

function common.CheckMissingElement(arr, cmp)
  local is_missing = {}
  local ok = true
  for _, key in ipairs(arr) do
    if cmp[key] == nil then
      table.insert(is_missing, key)
      ok = false
    end
  end
  return ok, is_missing
end

return common