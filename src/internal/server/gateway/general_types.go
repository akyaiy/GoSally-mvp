package gateway

import (
	"context"
	"net/http"

	"github.com/akyaiy/GoSally-mvp/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/internal/engine/app"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
	"github.com/akyaiy/GoSally-mvp/internal/server/session"
)

// serversApiVer is a type alias for string, used to represent API version strings in the GeneralServer.
type serversApiVer string

type ServerApiContract interface {
	GetVersion() string
	Handle(ctx context.Context, sid string, r *http.Request, req *rpc.RPCRequest) *rpc.RPCResponse
}

// GeneralServer implements the GeneralServerApiContract and serves as a router for different API versions.
type GatewayServer struct {
	// servers holds the registered servers by their API version.
	// The key is the version string, and the value is the server implementing GeneralServerApi
	servers map[serversApiVer]ServerApiContract

	sm *session.SessionManager
	cs *corestate.CoreState
	x  *app.AppX
}
