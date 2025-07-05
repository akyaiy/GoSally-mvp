package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"

	"golang.org/x/net/netutil"

	"github.com/akyaiy/GoSally-mvp/core/config"
	gs "github.com/akyaiy/GoSally-mvp/core/general_server"
	"github.com/akyaiy/GoSally-mvp/core/logs"
	"github.com/akyaiy/GoSally-mvp/core/sv1"
	"github.com/akyaiy/GoSally-mvp/core/update"

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
	updater := update.NewUpdater(*log, cfg)
	versuion, versionType, _ := updater.GetCurrentVersion()
	fmt.Printf("Current version: %s (%s)\n", versuion, versionType)
	ver, vert, _ := updater.GetLatestVersion(versionType)
	fmt.Printf("Latest version: %s (%s)\n", ver, vert)

	fmt.Println("Checking for updates...")
	isNewUpdate, _ := updater.CkeckUpdates()
	fmt.Println("Update check result:", isNewUpdate)
	serverv1 := sv1.InitV1Server(&sv1.HandlerV1InitStruct{
		Log:            *log,
		Config:         cfg,
		AllowedCmd:     regexp.MustCompile(`^[a-zA-Z0-9]+$`),
		ListAllowedCmd: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
		Ver:            "v1",
	})
	s := gs.InitGeneral(&gs.GeneralServerInit{
		Log:    *log,
		Config: cfg,
	}, serverv1)

	r := chi.NewRouter()
	r.Route(config.GetServerConsts().GetApiRoute()+config.GetServerConsts().GetComDirRoute(), func(r chi.Router) {
		r.Get("/", s.HandleList)
		r.Get("/{cmd}", s.Handle)
	})
	r.Route("/favicon.ico", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})

	address := cfg.Address
	if cfg.TlsEnabled {
		log.Info("HTTPS server started with TLS", slog.String("address", address))
		listener, err := net.Listen("tcp", address)
		if err != nil {
			log.Error("Failed to start TLS listener", slog.String("error", err.Error()))
			return
		}
		limitedListener := netutil.LimitListener(listener, 100)
		err = http.ServeTLS(limitedListener, r, cfg.CertFile, cfg.KeyFile)
		if err != nil {
			log.Error("Failed to start HTTPS server", slog.String("error", err.Error()))
		}
	} else {
		log.Info("HTTP server started", slog.String("address", address))
		listener, err := net.Listen("tcp", address)
		if err != nil {
			log.Error("Failed to start listener", slog.String("error", err.Error()))
			return
		}
		limitedListener := netutil.LimitListener(listener, 100)
		err = http.Serve(limitedListener, r)
		if err != nil {
			log.Error("Failed to start HTTP server", slog.String("error", err.Error()))
		}
	}
}
