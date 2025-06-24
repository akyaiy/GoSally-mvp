package main

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/akyaiy/GoSally-mvp/config"
	gs "github.com/akyaiy/GoSally-mvp/general_server"
	"github.com/akyaiy/GoSally-mvp/logs"
	"github.com/akyaiy/GoSally-mvp/sv1"

	"github.com/go-chi/chi/v5"
)

var log *slog.Logger
var cfg *config.ConfigConf

func init() {
	cfg = config.MustLoadConfig()

	log = logs.SetupLogger(cfg.Mode)
	log = log.With("mode", cfg.Mode)

	log.Info("Initializing server", slog.String("address", cfg.HTTPServer.Address))
	log.Debug("Server running in debug mode")
}

func main() {
	serverv1 := sv1.InitV1Server(&sv1.HandlerV1InitStruct{
		Log:            *logs.SetupLogger(cfg.Mode),
		Config:         cfg,
		AllowedCmd:     regexp.MustCompile(`^[a-zA-Z0-9]+$`),
		ListAllowedCmd: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
		Ver:            "v1",
	})
	s := gs.InitGeneral(&gs.GeneralServerInit{
		Log:    *logs.SetupLogger(cfg.Mode),
		Config: cfg,
	}, serverv1)
	r := chi.NewRouter()
	r.Route("/api/{ver}/com", func(r chi.Router) {
		r.Get("/", s.HandleList)
		r.Get("/{cmd}", s.Handle)
	})
	r.NotFound(serverv1.ErrNotFound)
	if cfg.TlsEnabled == "true" {
		log.Info("Server started with TLS", slog.String("address", cfg.Address))
		err := http.ListenAndServeTLS(cfg.Address, cfg.CertFile, cfg.KeyFile, r)
		if err != nil {
			log.Error("Failed to start HTTPS server", slog.String("error", err.Error()))
		}
	}
	log.Info("Server started", slog.String("address", cfg.Address))
	err := http.ListenAndServe(cfg.Address, r)
	if err != nil {
		log.Error("Failed to start HTTP server", slog.String("error", err.Error()))
	}
}
