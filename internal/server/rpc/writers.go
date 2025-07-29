package rpc

import (
	"encoding/json"
	"net/http"
)

func write(w http.ResponseWriter, msg *RPCResponse) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func WriteError(w http.ResponseWriter, errm *RPCResponse) error {
	return write(w, errm)
}

func WriteResponse(w http.ResponseWriter, response *RPCResponse) error {
	return write(w, response)
}
