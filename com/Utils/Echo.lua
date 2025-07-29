function validate()
	if not In.Params.msg or In.Params.msg == "" then
		Out.Error = {
			message = "there must be a msg parameter"
		}
		return
	end
end


validate()
Out.Result.answer = In.Params.msg