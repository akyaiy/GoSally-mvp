package general_server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/akyaiy/GoSally-mvp/config"

	"github.com/go-chi/chi/v5"
)

type serversApiVer string

type GeneralServerApiContract interface {
	GetVersion() string

	Handle(w http.ResponseWriter, r *http.Request)
	HandleList(w http.ResponseWriter, r *http.Request)
}

type GeneralServerContarct interface {
	GeneralServerApiContract
	AppendToArray(GeneralServerApiContract) error
}

type GeneralServer struct {
	w http.ResponseWriter
	r *http.Request

	servers map[serversApiVer]GeneralServerApiContract

	log slog.Logger
	cfg *config.ConfigConf
}

// structure only for initialization
type GeneralServerInit struct {
	Log    slog.Logger
	Config *config.ConfigConf
}

func InitGeneral(o *GeneralServerInit, servers ...GeneralServerApiContract) *GeneralServer {
	general := &GeneralServer{
		servers: make(map[serversApiVer]GeneralServerApiContract),
		cfg:     o.Config,
		log:     o.Log,
	}
	for _, s := range servers {
		general.servers[serversApiVer(s.GetVersion())] = s
	}
	return general
}

func (s *GeneralServer) GetVersion() string {
	return "general"
}

func (s *GeneralServer) AppendToArray(server GeneralServerApiContract) error {
	if _, exist := s.servers[serversApiVer(server.GetVersion())]; !exist {
		s.servers[serversApiVer(server.GetVersion())] = server
		return nil
	}
	return errors.New("server with this version is already exist")
}

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

	s.log.Info("Received request")

	if srv, ok := s.servers[serversApiVer(serverReqApiVer)]; ok {
		srv.Handle(w, r)
		return
	}

	if slices.Contains(s.cfg.Layers, serverReqApiVer) {
		if srv, ok := s.servers[serversApiVer(s.cfg.LatestVer)]; ok {
			s.log.Info("Using latest version under custom layer",
				slog.String("layer", serverReqApiVer),
				slog.String("fallback-version", s.cfg.LatestVer),
			)
			srv.Handle(w, r)
			return
		}
	}

	log.Error("HTTP request error: unsupported API version",
		slog.Int("status", http.StatusBadRequest))
	s.writeJSONError(http.StatusBadRequest, "unsupported API version")
}

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

	log.Info("Received request")

	if srv, ok := s.servers[serversApiVer(serverReqApiVer)]; ok {
		srv.HandleList(w, r)
		return
	}

	if slices.Contains(s.cfg.Layers, serverReqApiVer) {
		if srv, ok := s.servers[serversApiVer(s.cfg.LatestVer)]; ok {
			log.Info("Using latest version under custom layer",
				slog.String("layer", serverReqApiVer),
				slog.String("fallback-version", s.cfg.LatestVer),
			)
			srv.HandleList(w, r)
			return
		}
	}

	log.Error("HTTP request error: unsupported API version",
		slog.Int("status", http.StatusBadRequest))
	s.writeJSONError(http.StatusBadRequest, "unsupported API version")
}

// func (s *GeneralServer) _errNotFound() {
// 	s.writeJSONError(http.StatusBadRequest, "invalid request")
// 	s.log.Error("HTTP request error",
// 		slog.String("remote", s.r.RemoteAddr),
// 		slog.String("method", s.r.Method),
// 		slog.String("url", s.r.URL.String()),
// 		slog.Int("status", http.StatusBadRequest))
// }

func (s *GeneralServer) writeJSONError(status int, msg string) {
	s.w.Header().Set("Content-Type", "application/json")
	s.w.WriteHeader(status)
	resp := map[string]interface{}{
		"status": "error",
		"error":  msg,
		"code":   status,
	}
	json.NewEncoder(s.w).Encode(resp)
}
