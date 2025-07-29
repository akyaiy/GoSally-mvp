package gateway

import (
	"log/slog"
	"net/http"

	"github.com/akyaiy/GoSally-mvp/internal/engine/config"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
)

// serversApiVer is a type alias for string, used to represent API version strings in the GeneralServer.
type serversApiVer string

type ServerApiContract interface {
	GetVersion() string
	Handle(r *http.Request, req *rpc.RPCRequest) *rpc.RPCResponse
}

// GeneralServer implements the GeneralServerApiContract and serves as a router for different API versions.
type GatewayServer struct {
	// servers holds the registered servers by their API version.
	// The key is the version string, and the value is the server implementing GeneralServerApi
	servers map[serversApiVer]ServerApiContract

	log *slog.Logger
	cfg *config.Conf
}
