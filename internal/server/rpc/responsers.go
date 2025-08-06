package rpc

import "encoding/json"

func NewError(code int, message string, data any, id *json.RawMessage) *RPCResponse {
	if data != nil {
		return &RPCResponse{
			JSONRPC: JSONRPCVersion,
			ID:      id,
			Error: map[string]any{
				"code":    code,
				"message": message,
				"data":    data,
			},
		}
	}
	return &RPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

func NewResponse(result any, id *json.RawMessage) *RPCResponse {
	if result == nil {
		return &RPCResponse{
			JSONRPC: JSONRPCVersion,
			ID:      id,
		}
	}
	return &RPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  result,
	}
}
