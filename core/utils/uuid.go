package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/akyaiy/GoSally-mvp/core/config"
)

func NewUUID() (string, error) {
	bytes := make([]byte, int(config.GetInternalConsts().GetUUIDLength()/2))
	_, err := rand.Read(bytes)
	if err != nil {
		return "", errors.New("failed to generate UUID: " + err.Error())
	}
	return hex.EncodeToString(bytes), nil
}
