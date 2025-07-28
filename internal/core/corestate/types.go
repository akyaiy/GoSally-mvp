package corestate

// CoreStateContract is interface for CoreState.
// CoreState is a structure that contains the basic meta-information vital to the node.
// The interface contains functionality for working with the Runtime directory and its files,
// and access to low-level logging in stdout
type CoreStateContract interface {
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
}
