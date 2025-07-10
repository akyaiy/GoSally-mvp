package corestate

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/akyaiy/GoSally-mvp/core/config"
	"github.com/akyaiy/GoSally-mvp/core/utils"
)

// GetNodeUUID outputs the correct uuid from the file at the path specified in the arguments.
// If the uuid is not correct or is not exist, an empty string and an error will be returned.
// The path to the identifier must contain the path to the "uuid" directory,
// not the file with the identifier itself, for example: "uuid/data"
func GetNodeUUID(metaInfPath string) (string, error) {
	uuid, err := readNodeUUIDRaw(filepath.Join(metaInfPath, "data"))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(uuid[:]), nil
}

func readNodeUUIDRaw(p string) ([]byte, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return data, err
	}
	if len(data) != config.UUIDLength {
		return data, errors.New("decoded UUID length mismatch")
	}
	return data, nil
}

// SetNodeUUID sets the identifier to the given path.
// The function replaces the identifier's associated directory with all its contents.
func SetNodeUUID(metaInfPath string) error {
	if !strings.HasSuffix(metaInfPath, "uuid") {
		return errors.New("invalid meta/uuid path")
	}
	info, err := os.Stat(metaInfPath)
	if err == nil && info.IsDir() {
		err = os.RemoveAll(metaInfPath)
		if err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	err = os.MkdirAll(metaInfPath, 0755)
	if err != nil {
		return err
	}
	dataPath := filepath.Join(metaInfPath, "data")

	uuidStr, err := utils.NewUUID32Raw()
	if err != nil {
		return err
	}

	err = os.WriteFile(dataPath, uuidStr[:], 0644)
	if err != nil {
		return err
	}

	readmePath := filepath.Join(metaInfPath, "README.txt")
	readmeContent := ` - - - - ! STRICTLY FORBIDDEN TO MODIFY THIS DIRECTORY ! - - - - 
This directory contains the unique node identifier stored in the file named data.
This identifier is critical for correct node recognition both locally and across the network.
Any modification, deletion, or tampering with this directory may lead to permanent loss of identity, data corruption, or network conflicts.
Proceed at your own risk. You have been warned.`
	err = os.WriteFile(readmePath, []byte(readmeContent), 0644)
	if err != nil {
		return err
	}
	return nil
}
