package.path = "./com/?.lua;" .. package.path

print = function() end
io.write = function(...) end
io.stdout = function() return nil end
io.stderr = function() return nil end
io.read = function(...) return nil end