--- #description = "Echoes back the message provided in the 'msg' parameter. b1"

if not Params.msg then
    Result.status = "error"
    Result.error = "Missing parameter: msg"
    return
end

Result.status = "okv2"
Result.answer = Params.msg
return