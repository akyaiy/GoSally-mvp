package rpc

type RPCRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      any              `json:"id"`
	Method  string           `json:"method"`
	Params  RPCRequestParams `json:"params"`
}

type RPCRequestParams struct {
	ContextVersion string         `json:"context-version"`
	Method         map[string]any `json:"method-params"`
}

type RPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result"`
}

type RPCError struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Error   any    `json:"error"`
}

const (
	JSONRPCVersion = "2.0"
)
