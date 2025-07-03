package config

var UUIDLength int = 4

var ApiRoute string = "/api/{ver}"
var ComDirRoute string = "/com"

var NodeVersion string
var ActualFileNanme string = "actual.txt"

type _internalConsts struct{}
type _serverConsts struct{}
type _updateConsts struct{}

func GetUpdateConsts() _updateConsts { return _updateConsts{} }
func (_ _updateConsts) GetNodeVersion() string {
	if NodeVersion == "" {
		return "version0.0.0-none"
	}
	return NodeVersion
}
func (_ _updateConsts) GetActualFileName() string { return ActualFileNanme }

func GetInternalConsts() _internalConsts     { return _internalConsts{} }
func (_ _internalConsts) GetUUIDLength() int { return UUIDLength }

func GetServerConsts() _serverConsts           { return _serverConsts{} }
func (_ _serverConsts) GetApiRoute() string    { return ApiRoute }
func (_ _serverConsts) GetComDirRoute() string { return ComDirRoute }
