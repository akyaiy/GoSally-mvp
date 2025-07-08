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

type _internalConsts struct{}
type _serverConsts struct{}
type _updateConsts struct{}

func GetUpdateConsts() _updateConsts { return _updateConsts{} }
func (_ _updateConsts) GetNodeVersion() string {
	if NodeVersion == "" {
		return "v0.0.0-none"
	}
	return NodeVersion
}
func (_ _updateConsts) GetActualFileName() string     { return ActualFileName }
func (_ _updateConsts) GetUpdateArchiveName() string  { return UpdateArchiveName }
func (_ _updateConsts) GetUpdateDownloadPath() string { return UpdateDownloadPath }

func GetInternalConsts() _internalConsts     { return _internalConsts{} }
func (_ _internalConsts) GetUUIDLength() int { return UUIDLength }
func (_ _internalConsts) GetMetaDir() string { return MetaDir }

func GetServerConsts() _serverConsts           { return _serverConsts{} }
func (_ _serverConsts) GetApiRoute() string    { return ApiRoute }
func (_ _serverConsts) GetComDirRoute() string { return ComDirRoute }
