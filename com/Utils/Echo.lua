if In.Params and In.Params.about then
  Out.Result = {
    description = "Echo of the message",
    params = {
      msg = "just message"
    }
  }
  return
end

local function validate()
	if not In.Params.msg or In.Params.msg == "" then
		Out.Error = {
			message = "there must be a msg parameter"
		}
		return
	end
end


validate()
Out.Result.answer = In.Params.msg