--- #description = "Echoes back the message provided in the 'msg' parameter."

local mod = require("_for_echo")

if not Params.msg then
    Result.status = "error"
    Result.error = "Missing parameter: msg"
    return
end

Result.status = "ok"
Result.answer = mod.translate(Params.msg)
return