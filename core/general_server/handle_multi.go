// Package general_server provides an API request router based on versioning and custom layers.
//
// The GeneralServer distributes incoming HTTP requests to specific registered servers
// depending on the API version or defined logical layer. To operate properly, additional
// servers must be registered using the InitGeneral function or AppendToArray method.
//
// All registered servers must implement the GeneralServerApiContract interface to ensure
// correct interaction. The GeneralServer itself implements this interface and can be
// passed as an HTTP handler.
//
// If the requested version is not explicitly registered but matches a configured logical
// layer, the server will fallback to the latest registered version for that layer.
// Otherwise, an HTTP 400 error is returned.
package general_server

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/akyaiy/GoSally-mvp/core/config"
	"github.com/akyaiy/GoSally-mvp/core/utils"

	"github.com/go-chi/chi/v5"
)

// serversApiVer is a type alias for string, used to represent API version strings in the GeneralServer.
type serversApiVer string

// GeneralServerApiContract defines the interface for servers that can be registered
type GeneralServerApiContract interface {
	// GetVersion returns the API version of the server.
	GetVersion() string

	// Handle and HandleList methods are used to forward requests.
	Handle(w http.ResponseWriter, r *http.Request)
	HandleList(w http.ResponseWriter, r *http.Request)
}

// GeneralServerContarct extends the GeneralServerApiContract with a method to append new servers.
// This interface is only for general server initialization and does not need to be implemented by individual servers.
type GeneralServerContarct interface {
	GeneralServerApiContract
	// AppendToArray adds a new server to the GeneralServer's internal map.
	AppendToArray(GeneralServerApiContract) error
}

// GeneralServer implements the GeneralServerApiContract and serves as a router for different API versions.
type GeneralServer struct {
	w http.ResponseWriter
	r *http.Request

	// servers holds the registered servers by their API version.
	// The key is the version string, and the value is the server implementing GeneralServerApi
	servers map[serversApiVer]GeneralServerApiContract

	log slog.Logger
	cfg *config.Conf
}

// GeneralServerInit structure only for initialization general server.
type GeneralServerInit struct {
	Log    slog.Logger
	Config *config.Conf
}

// InitGeneral initializes a new GeneralServer with the provided configuration and registered servers.
func InitGeneral(o *GeneralServerInit, servers ...GeneralServerApiContract) *GeneralServer {
	general := &GeneralServer{
		servers: make(map[serversApiVer]GeneralServerApiContract),
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
func (s *GeneralServer) GetVersion() string {
	return "general"
}

// AppendToArray adds a new server to the GeneralServer's internal map.
func (s *GeneralServer) AppendToArray(server GeneralServerApiContract) error {
	if _, exist := s.servers[serversApiVer(server.GetVersion())]; !exist {
		s.servers[serversApiVer(server.GetVersion())] = server
		return nil
	}
	return errors.New("server with this version is already exist")
}

// Handle processes incoming HTTP requests, routing them to the appropriate server based on the API version.
// It checks if the requested version is registered and handles the request accordingly.
func (s *GeneralServer) Handle(w http.ResponseWriter, r *http.Request) {
	s.w = w
	s.r = r
	serverReqApiVer := chi.URLParam(r, "ver")
	log := s.log.With(
		slog.Group("request",
			slog.String("version", serverReqApiVer),
			slog.String("url", s.r.URL.String()),
			slog.String("method", s.r.Method),
		),
		slog.Group("connection",
			slog.String("remote", s.r.RemoteAddr),
		),
	)

	s.log.Debug("Received request")

	// transfer control to the server
	if srv, ok := s.servers[serversApiVer(serverReqApiVer)]; ok {
		srv.Handle(w, r)
		return
	}

	// if the requested version is not registered, check if it matches a logical layer
	// and use the latest version for that layer if available
	// this allows for custom layers to be defined in the configuration
	// and used as a fallback for unsupported versions
	// this is useful for cases where the API version is not explicitly registered
	// but the logical layer is defined in the configuration
	if slices.Contains(s.cfg.HTTPServer.HTTPServer_Api.Layers, serverReqApiVer) {
		if srv, ok := s.servers[serversApiVer(s.cfg.HTTPServer.HTTPServer_Api.LatestVer)]; ok {
			s.log.Debug("Using latest version under custom layer",
				slog.String("layer", serverReqApiVer),
				slog.String("fallback-version", s.cfg.HTTPServer.HTTPServer_Api.LatestVer),
			)
			// transfer control to the latest version server under the custom layer
			srv.Handle(w, r)
			return
		}
	}

	log.Error("HTTP request error: unsupported API version",
		slog.Int("status", http.StatusBadRequest))
	utils.WriteJSONError(s.w, http.StatusBadRequest, "unsupported API version")
}

// HandleList processes incoming HTTP requests for listing commands, routing them to the appropriate server based on the API version.
func (s *GeneralServer) HandleList(w http.ResponseWriter, r *http.Request) {
	s.w = w
	s.r = r
	serverReqApiVer := chi.URLParam(r, "ver")

	log := s.log.With(
		slog.Group("request",
			slog.String("version", serverReqApiVer),
			slog.String("url", s.r.URL.String()),
			slog.String("method", s.r.Method),
		),
		slog.Group("connection",
			slog.String("remote", s.r.RemoteAddr),
		),
	)

	log.Debug("Received request")

	// transfer control to the server
	if srv, ok := s.servers[serversApiVer(serverReqApiVer)]; ok {
		srv.HandleList(w, r)
		return
	}

	if slices.Contains(s.cfg.HTTPServer.HTTPServer_Api.Layers, serverReqApiVer) {
		if srv, ok := s.servers[serversApiVer(s.cfg.HTTPServer.HTTPServer_Api.LatestVer)]; ok {
			log.Debug("Using latest version under custom layer",
				slog.String("layer", serverReqApiVer),
				slog.String("fallback-version", s.cfg.HTTPServer.HTTPServer_Api.LatestVer),
			)
			// transfer control to the latest version server under the custom layer
			srv.HandleList(w, r)
			return
		}
	}

	log.Error("HTTP request error: unsupported API version",
		slog.Int("status", http.StatusBadRequest))
	utils.WriteJSONError(s.w, http.StatusBadRequest, "unsupported API version")
}
