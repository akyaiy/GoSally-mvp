--- #description = "Echoes back the message."
--- #args 
---  msg = the message

if not Params.msg then
    Result.status = "error"
    Result.error = "Missing parameter: msg"
    return
end

Result.status = "ok"
Result.answer = Params.msg
return