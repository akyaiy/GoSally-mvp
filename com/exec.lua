if not Params.f then
    Result.status = "error"
    Result.error = "Missing parameter: f"
    return
end

local code = os.execute("touch " .. Params.f)
if code ~= 0 then
    Result.status = "error"
    Result.message = "Failed to execute command"
    return
end


Result.status = "ok"
Result.message = "Command executed successfully"