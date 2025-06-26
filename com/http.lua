package.path = package.path .. ";/usr/lib64/lua/5.1/?.lua;/usr/local/share/lua/5.1/?.lua;" .. ";./com/?.lua;"
package.cpath = package.cpath .. ";/usr/lib64/lua/5.1/?.so;/usr/local/lib/lua/5.1/?.so;"

local https = require("ssl.https")
local ltn12 = require("ltn12")

local response = {}
local res, code, headers = https.request{
    url = "https://localhost:8080/api/v1/echo?msg=sigma",
    sink = ltn12.sink.table(response)
}


Result.msg = table.concat(response)
Result.status = "ok"
