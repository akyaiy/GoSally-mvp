package run_manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/akyaiy/GoSally-mvp/internal/core/utils"
)

type RunManagerContract interface {
	Get(index string) (string, error)

	// Set recursively creates a file in runDir
	Set(index string) error

	File(index string) RunFileManagerContract

	indexPaths() error
}

var (
	created      bool
	runDir       string
	indexedPaths = make(map[string]string)
)

type RunFileManagerContract interface {
	Open() (*os.File, error)
	Close() error
	Watch(parentCtx context.Context, callback func()) (context.CancelFunc, error)
}

type RunFileManager struct {
	err         error
	indexedPath string
	file        *os.File
}

// func (c *CoreState) RuntimeDir() RunManagerContract {
// 	return c.RM
// }

// Create creates a temp directory
func Create(uuid32 string) (string, error) {
	if created {
		return runDir, fmt.Errorf("runtime directory is already created")
	}
	path, err := os.MkdirTemp("", fmt.Sprintf("*-%s-%s", uuid32, "gosally-runtime"))
	if err != nil {
		return "", err
	}
	runDir = path
	created = true
	return path, nil
}

func Clean() error {
	created = false
	indexedPaths = nil
	return utils.CleanTempRuntimes(runDir)
}

// Quite dangerous and goofy.
// TODO: implement a better variant of runDir indexing on the second stage of initialization
func Toggle() string {
	runDir = filepath.Dir(os.Args[0])
	created = true
	return runDir
}

func Get(index string) (string, error) {
	if !created {
		return "", fmt.Errorf("runtime directory is not created")
	}
	if indexedPaths == nil {
		err := indexPaths()
		if err != nil {
			return "", nil
		}
	}
	if indexedPaths == nil {
		return "", fmt.Errorf("indexedPaths is nil")
	}
	value, ok := indexedPaths[index]
	if !ok {
		err := indexPaths()
		if err != nil {
			return "", err
		}
		value, ok = indexedPaths[index]
		if !ok {
			return "", fmt.Errorf("cannot detect file under index %s", index)
		}
	}
	return value, nil
}

func Set(index string) error {
	if !created {
		return fmt.Errorf("runtime directory is not created")
	}
	fullPath := filepath.Join(runDir, index)

	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if indexedPaths == nil {
		err = indexPaths()
		if err != nil {
			return err
		}
	} else {
		indexedPaths[index] = fullPath
	}

	return nil
}

func SetDir(index string) error {
	if !created {
		return fmt.Errorf("runtime directory is not created")
	}
	fullPath := filepath.Join(runDir, index)

	err := os.MkdirAll(fullPath, 0755)
	if err != nil {
		return err
	}

	return nil
}

func indexPaths() error {
	if !created {
		return fmt.Errorf("runtime directory is not created")
	}
	i, err := utils.IndexPaths(runDir)
	if err != nil {
		return err
	}
	indexedPaths = i
	return nil
}

func RuntimeDir() string {
	return runDir
}
