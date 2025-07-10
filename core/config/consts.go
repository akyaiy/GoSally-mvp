package config

import "os"

// UUIDLength is uuids length for sessions. By default it is 16 bytes.
var UUIDLength int = 16

// ApiRoute setting for go-chi for main route for api requests
var ApiRoute string = "/api/{ver}"

// ComDirRoute setting for go-chi for main route for commands
var ComDirRoute string = "/com"

// NodeVersion is the version of the node. It can be set by the build system or manually.
// If not set, it will return "version0.0.0-none" by default
var NodeVersion string

// ActualFileName is a feature of the GoSally update system.
// In the repository, the file specified in the variable contains the current information about updates
var ActualFileName string = "actual.txt"

// UpdateArchiveName is the name of the archive that will be used for updates.
var UpdateArchiveName string = "gosally-node"

// UpdateInstallPath is the path where the update will be installed.
var UpdateDownloadPath string = os.TempDir()

var MetaDir string = "./.meta"

func init() {
	if NodeVersion == "" {
		NodeVersion = "v0.0.0-none"
	}
}
