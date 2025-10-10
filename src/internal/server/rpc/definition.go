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
	Result  any              `json:"result,omitzero"`
	Error   any              `json:"error,omitzero"`
	Data    *RPCData         `json:"data,omitzero"`
}

type RPCData struct {
	ResponsibleNode string `json:"responsible-node,omitempty"`
	Salt            string `json:"salt,omitempty"`
	Checksum        string `json:"checksum-md5,omitempty"`
	NewSessionUUID  string `json:"new-session-uuid,omitempty"`
}

const (
	JSONRPCVersion = "2.0"
)
