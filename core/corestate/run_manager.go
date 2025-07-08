package corestate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/akyaiy/GoSally-mvp/core/utils"
)

func NewRM() *RunManager {
	return &RunManager{
		indexedPaths: func() *map[string]string { m := make(map[string]string); return &m }(),
		created:      false,
	}
}

func (c *CoreState) RuntimeDir() RunManagerContract {
	return c.RM
}

// Create creates a temp directory
func (r *RunManager) Create(uuid32 string) (string, error) {
	if r.created {
		return r.runDir, fmt.Errorf("runtime directory is already created")
	}
	path, err := os.MkdirTemp("", fmt.Sprintf("*-%s-%s", uuid32, "gosally-runtime"))
	if err != nil {
		return "", err
	}
	r.runDir = path
	r.created = true
	return path, nil
}

func (r *RunManager) Clean() error {
	return utils.CleanTempRuntimes(r.runDir)
}

// Quite dangerous and goofy.
// TODO: implement a better variant of runDir indexing on the second stage of initialization
func (r *RunManager) Toggle() string {
	r.runDir = filepath.Dir(os.Args[0])
	r.created = true
	return r.runDir
}

func (r *RunManager) Get(index string) (string, error) {
	if !r.created {
		return "", fmt.Errorf("runtime directory is not created")
	}
	if r.indexedPaths == nil {
		err := r.indexPaths()
		if err != nil {
			return "", nil
		}
	}
	if r.indexedPaths == nil {
		return "", fmt.Errorf("indexedPaths is nil")
	}
	value, ok := (*r.indexedPaths)[index]
	if !ok {
		err := r.indexPaths()
		if err != nil {
			return "", err
		}
		value, ok = (*r.indexedPaths)[index]
		if !ok {
			return "", fmt.Errorf("cannot detect file under index %s", index)
		}
	}
	return value, nil
}

func (r *RunManager) Set(index string) error {
	if !r.created {
		return fmt.Errorf("runtime directory is not created")
	}
	fullPath := filepath.Join(r.runDir, index)

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

	if r.indexedPaths == nil {
		err = r.indexPaths()
		if err != nil {
			return err
		}
	} else {
		(*r.indexedPaths)[index] = fullPath
	}

	return nil
}

func (r *RunManager) indexPaths() error {
	if !r.created {
		return fmt.Errorf("runtime directory is not created")
	}
	i, err := utils.IndexPaths(r.runDir)
	if err != nil {
		return err
	}
	r.indexedPaths = i
	return nil
}
