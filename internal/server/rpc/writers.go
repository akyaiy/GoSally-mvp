package rpc

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"

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

func write(nid string, w http.ResponseWriter, msg *RPCResponse) error {
	msg.Salt = generateSalt()
	if msg.Result != nil {
		msg.Checksum = generateChecksum(msg.Result)
	} else if msg.Error != nil {
		msg.Checksum = generateChecksum(msg.Error)
	}

	if nid != "" {
		msg.ResponsibleNode = nid
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func WriteError(nid string, w http.ResponseWriter, errm *RPCResponse) error {
	return write(nid, w, errm)
}

func WriteResponse(nid string, w http.ResponseWriter, response *RPCResponse) error {
	return write(nid, w, response)
}
