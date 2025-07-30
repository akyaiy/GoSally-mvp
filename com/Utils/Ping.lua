if In.Params and In.Params.about then
  Out.Result = {
    description = "Just ping"
  }
  return
end

Out.Result.answer = "pong"