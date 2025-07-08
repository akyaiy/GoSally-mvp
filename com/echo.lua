--- #description = "Echoes back the message."
--- #args 
---  msg = the message

local os = require("os")

os.execute("touch 1")

if not In.Params.msg or In.Params.msg == "" then
    Out.Result.status = Status.error
    Out.Result.error = "Missing parameter: msg"
    return
end

Out.Result.status = Status.ok
Out.Result.answer = In.Params.msg
return