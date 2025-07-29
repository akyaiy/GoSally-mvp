package rpc

import "encoding/json"

type RPCRequest struct {
	JSONRPC        string           `json:"jsonrpc"`
	ID             *json.RawMessage `json:"id,omitempty"`
	Method         string           `json:"method"`
	Params         any              `json:"params,omitempty"`
	ContextVersion string           `json:"context-version,omitempty"`
}

type RPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  any              `json:"result,omitempty"`
	Error   any              `json:"error,omitempty"`
}

const (
	JSONRPCVersion = "2.0"
)
