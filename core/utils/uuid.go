package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/akyaiy/GoSally-mvp/core/config"
)

func NewUUIDRaw(length int) ([]byte, error) {
	bytes := make([]byte, int(length))
	_, err := rand.Read(bytes)
	if err != nil {
		return bytes, errors.New("failed to generate UUID: " + err.Error())
	}
	return bytes, nil
}

func NewUUID(length int) (string, error) {
	data, err := NewUUIDRaw(length)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func NewUUID32() (string, error) {
	return NewUUID(config.GetInternalConsts().GetUUIDLength())
}

func NewUUID32Raw() ([]byte, error) {
	data, err := NewUUIDRaw(config.GetInternalConsts().GetUUIDLength())
	if err != nil {
		return data, err
	}
	if len(data) != config.GetInternalConsts().GetUUIDLength() {
		return data, errors.New("unexpected UUID length")
	}
	return data, nil
}
