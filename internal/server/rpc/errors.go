package rpc

const (
	ErrParseError  = -32700
	ErrParseErrorS = "Parse error"

	ErrInvalidRequest  = -32600
	ErrInvalidRequestS = "Invalid Request"

	ErrMethodNotFound  = -32601
	ErrMethodNotFoundS = "Method not found"

	ErrInvalidParams  = -32602
	ErrInvalidParamsS = "Invalid params"

	ErrInternalError  = -32603
	ErrInternalErrorS = "Internal error"

	ErrContextVersion  = -32010
	ErrContextVersionS = "Invalid context version"

	ErrInvalidMethodFormat  = -32020
	ErrInvalidMethodFormatS = "Invalid method format"

	ErrMethodIsMissing  = -32020
	ErrMethodIsMissingS = "Method is missing"

	ErrSessionIsTaken  = -32030
	ErrSessionIsTakenS = "The session is already taken"
)
