package corestate

import (
	"context"
	"os"
)

// CoreStateContract is interface for CoreState.
// CoreState is a structure that contains the basic meta-information vital to the node.
// The interface contains functionality for working with the Runtime directory and its files,
// and access to low-level logging in stdout
type CoreStateContract interface {
	RuntimeDir() RunManagerContract
}

type CoreState struct {
	UUID32        string
	UUID32DirName string

	StartTimestampUnix int64

	NodeBinName string
	NodeVersion string

	Stage Stage

	NodePath string
	MetaDir  string
	RunDir   string

	RM *RunManager
}

type RunManagerContract interface {
	Get(index string) (string, error)

	// Set recursively creates a file in runDir
	Set(index string) error

	File(index string) RunFileManagerContract

	indexPaths() error
}

type RunManager struct {
	created bool
	runDir  string
	// I obviously keep it with a pointer because it makes me feel calmer
	indexedPaths *map[string]string
}

type RunFileManagerContract interface {
	Open() (*os.File, error)
	Close() error
	Watch(ctx context.Context, callback func()) error
}

type RunFileManager struct {
	err         error
	indexedPath string
	file        *os.File
}
