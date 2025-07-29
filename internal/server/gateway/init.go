package gateway

import (
	"errors"
	"log/slog"

	"github.com/akyaiy/GoSally-mvp/internal/engine/config"
)

// GeneralServerInit structure only for initialization general server.
type GatewayServerInit struct {
	Log    *slog.Logger
	Config *config.Conf
}

// InitGeneral initializes a new GeneralServer with the provided configuration and registered servers.
func InitGateway(o *GatewayServerInit, servers ...ServerApiContract) *GatewayServer {
	general := &GatewayServer{
		servers: make(map[serversApiVer]ServerApiContract),
		cfg:     o.Config,
		log:     o.Log,
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
