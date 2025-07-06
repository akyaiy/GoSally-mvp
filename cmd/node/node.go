package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"time"

	"golang.org/x/net/netutil"

	"github.com/akyaiy/GoSally-mvp/core/config"
	gs "github.com/akyaiy/GoSally-mvp/core/general_server"
	_ "github.com/akyaiy/GoSally-mvp/core/init"
	"github.com/akyaiy/GoSally-mvp/core/logs"
	"github.com/akyaiy/GoSally-mvp/core/sv1"
	"github.com/akyaiy/GoSally-mvp/core/update"
	"github.com/go-chi/cors"

	"github.com/go-chi/chi/v5"
)

var log *slog.Logger
var cfg *config.ConfigConf

func init() {
	cfg = config.MustLoadConfig()

	log = logs.SetupLogger(cfg.Mode)
	log = log.With("mode", cfg.Mode)

	currentV, currentB, _ := update.NewUpdater(*log, cfg).GetCurrentVersion()

	log.Info("Initializing GoSally server", slog.String("address", cfg.HTTPServer.Address), slog.String("version", string(currentV)+"-"+string(currentB)))
	log.Debug("Server running in debug mode")
}

func UpdateDaemon(u *update.Updater, cfg config.ConfigConf, srv *http.Server) {
	//time.Sleep(5 * time.Second)
	log.Info("New update available, starting update process...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("Trying to down server gracefully before update")
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Failed to shutdown server gracefully", slog.String("error", err.Error()))
	}

	err := u.Update()
	if err != nil {
		log.Error("Failed to update", slog.String("error", err.Error()))
	} else {
		log.Info("Update completed successfully")
	}

}

func main() {
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
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Route(config.GetServerConsts().GetApiRoute()+config.GetServerConsts().GetComDirRoute(), func(r chi.Router) {
		r.Get("/", s.HandleList)
		r.Get("/{cmd}", s.Handle)
	})
	r.Route("/favicon.ico", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})

	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: r,
	}
	go func() {
		if cfg.TlsEnabled {
			log.Info("HTTPS server started with TLS", slog.String("address", cfg.Address))
			listener, err := net.Listen("tcp", cfg.Address)
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
			log.Info("HTTP server started", slog.String("address", cfg.Address))
			listener, err := net.Listen("tcp", cfg.Address)
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
	}()

	time.Sleep(5*time.Second)
	updater := update.NewUpdater(*log, cfg)
	go func() {
		time.Sleep(6*time.Second)
		for {
			isNewUpdate, err := updater.CkeckUpdates()
			if err != nil {
				log.Error("Failed to check for updates", slog.String("error", err.Error()))
			}
			if isNewUpdate {
				UpdateDaemon(updater, *cfg, srv)
			} else {
				log.Info("No new updates available")
			}
			time.Sleep(cfg.CheckInterval)
		}
	}()

	select {}
}
