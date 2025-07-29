package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func write(w http.ResponseWriter, status int, msg any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	switch m := msg.(type) {
	case RPCError, *RPCError,
		RPCResponse, *RPCResponse:
		data, err := json.Marshal(m)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		return fmt.Errorf("invalid RPC structure: %T", msg)
	}
}

func WriteRouterError(w http.ResponseWriter, status int, errm *RPCError) error {
	return write(w, status, errm)
}

func WriteResponse(w http.ResponseWriter, response *RPCResponse) error {
	return write(w, http.StatusOK, response)
}
