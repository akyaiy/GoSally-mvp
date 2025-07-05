package config

// UUIDLength is uuids length for sessions. By default it is 16 bytes.
var UUIDLength byte = 4

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
func (_ _updateConsts) GetActualFileName() string { return ActualFileName }

func GetInternalConsts() _internalConsts     { return _internalConsts{} }
func (_ _internalConsts) GetUUIDLength() byte { return UUIDLength }

func GetServerConsts() _serverConsts           { return _serverConsts{} }
func (_ _serverConsts) GetApiRoute() string    { return ApiRoute }
func (_ _serverConsts) GetComDirRoute() string { return ComDirRoute }
