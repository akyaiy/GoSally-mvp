package hooks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/akyaiy/GoSally-mvp/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/internal/core/run_manager"
	"github.com/akyaiy/GoSally-mvp/internal/core/update"
	"github.com/akyaiy/GoSally-mvp/internal/core/utils"
	"github.com/akyaiy/GoSally-mvp/internal/engine/app"
	"github.com/akyaiy/GoSally-mvp/internal/engine/config"
	"github.com/akyaiy/GoSally-mvp/internal/engine/logs"
	"github.com/akyaiy/GoSally-mvp/internal/server/gateway"
	"github.com/akyaiy/GoSally-mvp/internal/server/session"
	"github.com/akyaiy/GoSally-mvp/internal/server/sv1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/spf13/cobra"
	"golang.org/x/net/netutil"
)

var nodeApp = app.New()

func Run(cmd *cobra.Command, args []string) {
	nodeApp.InitialHooks(
		Init0Hook, Init1Hook, Init2Hook,
		Init3Hook, Init4Hook, Init5Hook,
		Init6Hook, Init7Hook,
	)

	nodeApp.Run(RunHook)
}

func RunHook(ctx context.Context, cs *corestate.CoreState, x *app.AppX) error {
	ctxMain, cancelMain := context.WithCancel(ctx)
	runLockFile := run_manager.File("run.lock")
	_, err := runLockFile.Open()
	if err != nil {
		x.Log.Fatalf("cannot open run.lock: %s", err)
	}

	_, err = runLockFile.Watch(ctxMain, func() {
		x.Log.Printf("run.lock was touched")
		_ = run_manager.Clean()
		cancelMain()
	})
	if err != nil {
		x.Log.Printf("watch error: %s", err)
	}

	serverv1 := sv1.InitV1Server(&sv1.HandlerV1InitStruct{
		X:          x,
		CS:         cs,
		AllowedCmd: regexp.MustCompile(`^[a-zA-Z0-9]+(>[a-zA-Z0-9]+)*$`),
		Ver:        "v1",
	})

	session_manager := session.New(*x.Config.Conf.HTTPServer.SessionTTL)

	s := gateway.InitGateway(&gateway.GatewayServerInit{
		SM: session_manager,
		CS: cs,
		X:  x,
	}, serverv1)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.HandleFunc(config.ComDirRoute, s.Handle)
	r.Route("/favicon.ico", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})

	srv := &http.Server{
		Addr:    *x.Config.Conf.HTTPServer.Address,
		Handler: r,
		ErrorLog: log.New(&logs.SlogWriter{
			Logger: x.SLog,
			Level:  slog.LevelError,
		}, "", 0),
	}

	nodeApp.Fallback(func(ctx context.Context, cs *corestate.CoreState, x *app.AppX) {
		if err := srv.Shutdown(ctxMain); err != nil {
			x.Log.Printf("%s: Failed to stop the server gracefully: %s", logs.PrintError(), err.Error())
		} else {
			x.Log.Printf("Server stopped gracefully")
		}

		x.Log.Println("Cleaning up...")

		if err := run_manager.Clean(); err != nil {
			x.Log.Printf("%s: Cleanup error: %s", logs.PrintError(), err.Error())
		}
		x.Log.Println("bye!")
	})

	go func() {
		defer utils.CatchPanicWithCancel(cancelMain)
		if *x.Config.Conf.TLS.TlsEnabled {
			listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", *x.Config.Conf.HTTPServer.Address, *x.Config.Conf.HTTPServer.Port))
			if err != nil {
				x.Log.Printf("%s: Failed to start TLS listener: %s", logs.PrintError(), err.Error())
				cancelMain()
				return
			}
			x.Log.Printf("Serving on %s port %s with TLS... (https://%s%s)", *x.Config.Conf.HTTPServer.Address, *x.Config.Conf.HTTPServer.Port, fmt.Sprintf("%s:%s", *x.Config.Conf.HTTPServer.Address, *x.Config.Conf.HTTPServer.Port), config.ComDirRoute)
			limitedListener := netutil.LimitListener(listener, 100)
			if err := srv.ServeTLS(limitedListener, *x.Config.Conf.TLS.CertFile, *x.Config.Conf.TLS.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				x.Log.Printf("%s: Failed to start HTTPS server: %s", logs.PrintError(), err.Error())
				cancelMain()
			}
		} else {
			x.Log.Printf("Serving on %s port %s... (http://%s%s)", *x.Config.Conf.HTTPServer.Address, *x.Config.Conf.HTTPServer.Port, fmt.Sprintf("%s:%s", *x.Config.Conf.HTTPServer.Address, *x.Config.Conf.HTTPServer.Port), config.ComDirRoute)
			listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", *x.Config.Conf.HTTPServer.Address, *x.Config.Conf.HTTPServer.Port))
			if err != nil {
				x.Log.Printf("%s: Failed to start listener: %s", logs.PrintError(), err.Error())
				cancelMain()
				return
			}
			limitedListener := netutil.LimitListener(listener, 100)
			if err := srv.Serve(limitedListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				x.Log.Printf("%s: Failed to start HTTP server: %s", logs.PrintError(), err.Error())
				cancelMain()
			}
		}
	}()

	session_manager.StartCleanup(5 * time.Second)

	if *x.Config.Conf.Updates.UpdatesEnabled {
		go func() {
			defer utils.CatchPanicWithCancel(cancelMain)
			updated := update.NewUpdater(&update.UpdaterInit{
				X:      x,
				Ctx:    ctxMain,
				Cancel: cancelMain,
			})
			updated.Shutdownfunc(cancelMain)
			for {
				isNewUpdate, err := updated.CkeckUpdates()
				if err != nil {
					x.Log.Printf("Failed to check for updates: %s", err.Error())
				}
				if isNewUpdate {
					if err := updated.Update(); err != nil {
						x.Log.Printf("Failed to update: %s", err.Error())
					} else {
						x.Log.Printf("Update completed successfully")
					}
				}
				time.Sleep(*x.Config.Conf.Updates.CheckInterval)
			}
		}()
	}

	<-ctxMain.Done()
	nodeApp.CallFallback(ctx)
	return nil
}
