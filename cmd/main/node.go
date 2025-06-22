package main

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/akyaiy/GoSally-mvp/config"
	"github.com/akyaiy/GoSally-mvp/logs"
	"github.com/akyaiy/GoSally-mvp/v1"

	"github.com/go-chi/chi/v5"
)

var allowedCmd = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
var log *slog.Logger
var cfg *config.ConfigConf
var listAllowedCmd = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`) // allowed symbols after first symbol

func init() {
	cfg = config.MustLoadConfig()

	log = logs.SetupLogger(cfg.Mode)
	log = log.With("mode", cfg.Mode)

	log.Info("Initializing server", slog.String("address", cfg.HTTPServer.Address))
	log.Debug("Server running in debug mode")
}

func main() {

	r := chi.NewRouter()
	r.Route("/v1/com", func(r chi.Router) {
		r.Get("/", handleV1ComList)
		r.Get("/{cmd}", handleV1)
	})
	// r.Route("/v2/com", func(r chi.Router) {
	// 	r.Get("/", handleV1ComList)
	// 	r.Get("/{cmd}", handleV1)
	// })
	r.NotFound(notFound)
	log.Info("Server started", slog.String("address", cfg.Address))
	http.ListenAndServe(cfg.Address, r)

}
