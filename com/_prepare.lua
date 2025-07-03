---@diagnostic disable: duplicate-set-field
package.path = package.path .. ";/usr/lib64/lua/5.1/?.lua;/usr/local/share/lua/5.1/?.lua" .. ";./com/?.lua;"
package.cpath = package.cpath .. ";/usr/lib64/lua/5.1/?.so;/usr/local/lib/lua/5.1/?.so"

print = function() end
io.write = function(...) end
io.stdout = function() return nil end
io.stderr = function() return nil end
io.read = function(...) return nil end

---@type table<string, any>
Status = {
	ok = "ok",
	error = "error",
	invalid = "invalid",
}
