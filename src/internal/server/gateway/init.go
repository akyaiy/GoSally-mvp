package gateway

import (
	"errors"

	"github.com/akyaiy/GoSally-mvp/src/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/src/internal/engine/app"
	"github.com/akyaiy/GoSally-mvp/src/internal/server/session"
)

// GeneralServerInit structure only for initialization general server.
type GatewayServerInit struct {
	SM *session.SessionManager
	CS *corestate.CoreState
	X  *app.AppX
}

// InitGeneral initializes a new GeneralServer with the provided configuration and registered servers.
func InitGateway(o *GatewayServerInit, servers ...ServerApiContract) *GatewayServer {
	general := &GatewayServer{
		servers: make(map[serversApiVer]ServerApiContract),
		sm:      o.SM,
		cs:      o.CS,
		x:       o.X,
	}

	// register the provided servers
	// s is each server implementing GeneralServerApiContract, this is not a general server
	for _, s := range servers {
		general.servers[serversApiVer(s.GetVersion())] = s
	}
	return general
}

// GetVersion returns the API version of the GeneralServer, which is "general".
func (s *GatewayServer) GetVersion() string {
	return "general"
}

// AppendToArray adds a new server to the GeneralServer's internal map.
func (s *GatewayServer) AppendToArray(server ServerApiContract) error {
	if _, exist := s.servers[serversApiVer(server.GetVersion())]; !exist {
		s.servers[serversApiVer(server.GetVersion())] = server
		return nil
	}
	return errors.New("server with this version is already exist")
}
