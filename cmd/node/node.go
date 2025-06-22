package main

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/akyaiy/GoSally-mvp/config"
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
	})
	r := chi.NewRouter()
	r.Route("/v1/com", func(r chi.Router) {
		r.Get("/", serverv1.HandleList)
		r.Get("/{cmd}", serverv1.Handle)
	})
	// r.Route("/v2/com", func(r chi.Router) {
	// 	r.Get("/", handleV1ComList)
	// 	r.Get("/{cmd}", handleV1)
	// })
	r.NotFound(serverv1.ErrNotFound)
	log.Info("Server started", slog.String("address", cfg.Address))
	http.ListenAndServe(cfg.Address, r)

}
