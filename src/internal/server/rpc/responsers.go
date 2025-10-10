package rpc

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/akyaiy/GoSally-mvp/src/internal/core/corestate"
	"github.com/google/uuid"
)

func generateChecksum(result any) string {
	if result == nil {
		return ""
	}
	data, err := json.Marshal(result)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", md5.Sum(data))
}

func generateSalt() string {
	return uuid.NewString()
}

func GetData(data any) *RPCData {
	return &RPCData{
		Salt:            generateSalt(),
		ResponsibleNode: corestate.NODE_UUID,
		Checksum:        generateChecksum(data),
	}
}

func NewError(code int, message string, data any, id *json.RawMessage) *RPCResponse {
	Error := make(map[string]any)
	Error = map[string]any{
		"code":    code,
		"message": message,
	}
	if data != nil {
		Error["data"] = data
	}

	return &RPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error:   Error,
		Data:    GetData(Error),
	}
}

func NewResponse(result any, id *json.RawMessage) *RPCResponse {
	return &RPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  result,
		Data:    GetData(result),
	}
}
