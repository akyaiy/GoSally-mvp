package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/akyaiy/GoSally-mvp/core/app"
	"github.com/akyaiy/GoSally-mvp/core/config"
	"github.com/akyaiy/GoSally-mvp/core/corestate"
	gs "github.com/akyaiy/GoSally-mvp/core/general_server"
	"github.com/akyaiy/GoSally-mvp/core/logs"
	"github.com/akyaiy/GoSally-mvp/core/run_manager"
	"github.com/akyaiy/GoSally-mvp/core/sv1"
	"github.com/akyaiy/GoSally-mvp/core/update"
	"github.com/akyaiy/GoSally-mvp/core/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/spf13/cobra"
	"golang.org/x/net/netutil"
	"gopkg.in/ini.v1"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run node normally",
	Run: func(cmd *cobra.Command, args []string) {
		nodeApp := app.New()

		nodeApp.InitialHooks(
			func(cs *corestate.CoreState, x *app.AppX) {
				x.Config = compositor
				x.Log.SetOutput(os.Stdout)
				x.Log.SetPrefix(logs.SetBrightBlack(fmt.Sprintf("(%s) ", cs.Stage)))
				x.Log.SetFlags(log.Ldate | log.Ltime)
			},

			// First stage: pre-init
			func(cs *corestate.CoreState, x *app.AppX) {
				*cs = *corestate.NewCorestate(&corestate.CoreState{
					UUID32DirName:      "uuid",
					NodeBinName:        filepath.Base(os.Args[0]),
					NodeVersion:        config.NodeVersion,
					MetaDir:            "./.meta",
					Stage:              corestate.StagePreInit,
					StartTimestampUnix: time.Now().Unix(),
				})
			},

			func(cs *corestate.CoreState, x *app.AppX) {
				x.Log.SetPrefix(logs.SetBlue(fmt.Sprintf("(%s) ", cs.Stage)))

				if err := x.Config.LoadEnv(); err != nil {
					x.Log.Fatalf("env load error: %s", err)
				}
				cs.NodePath = x.Config.Env.NodePath

				if cfgPath := x.Config.CMDLine.Run.ConfigPath; cfgPath != "" {
					x.Config.Env.ConfigPath = cfgPath
				}
				if err := x.Config.LoadConf(x.Config.Env.ConfigPath); err != nil {
					x.Log.Fatalf("conf load error: %s", err)
				}
			},

			func(cs *corestate.CoreState, x *app.AppX) {
				uuid32, err := corestate.GetNodeUUID(filepath.Join(cs.MetaDir, "uuid"))
				if errors.Is(err, fs.ErrNotExist) {
					if err := corestate.SetNodeUUID(filepath.Join(cs.NodePath, cs.MetaDir, cs.UUID32DirName)); err != nil {
						x.Log.Fatalf("Cannod generate node uuid: %s", err.Error())
					}
					uuid32, err = corestate.GetNodeUUID(filepath.Join(cs.MetaDir, "uuid"))
					if err != nil {
						x.Log.Fatalf("Unexpected failure: %s", err.Error())
					}
				}
				if err != nil {
					x.Log.Fatalf("uuid load error: %s", err)
				}
				cs.UUID32 = uuid32
			},

			func(cs *corestate.CoreState, x *app.AppX) {
				if x.Config.Env.ParentStagePID != os.Getpid() {
					if os.TempDir() != "/tmp" {
						x.Log.Printf("%s: %s", logs.SetYellow("warning"), "non-standard value specified for temporary directory")
					}
					// still pre-init stage
					runDir, err := run_manager.Create(cs.UUID32)
					if err != nil {
						x.Log.Fatalf("Unexpected failure: %s", err.Error())
					}
					cs.RunDir = runDir
					input, err := os.Open(os.Args[0])
					if err != nil {
						run_manager.Clean()
						x.Log.Fatalf("Unexpected failure: %s", err.Error())
					}
					if err := run_manager.Set(cs.NodeBinName); err != nil {
						run_manager.Clean()
						x.Log.Fatalf("Unexpected failure: %s", err.Error())
					}
					fmgr := run_manager.File(cs.NodeBinName)
					output, err := fmgr.Open()
					if err != nil {
						run_manager.Clean()
						x.Log.Fatalf("Unexpected failure: %s", err.Error())
					}

					if _, err := io.Copy(output, input); err != nil {
						fmgr.Close()
						run_manager.Clean()
						x.Log.Fatalf("Unexpected failure: %s", err.Error())
					}
					if err := os.Chmod(filepath.Join(cs.RunDir, cs.NodeBinName), 0755); err != nil {
						fmgr.Close()
						run_manager.Clean()
						x.Log.Fatalf("Unexpected failure: %s", err.Error())
					}
					input.Close()
					fmgr.Close()
					runArgs := os.Args
					runArgs[0] = filepath.Join(cs.RunDir, cs.NodeBinName)

					// prepare environ
					env := utils.SetEviron(os.Environ(), fmt.Sprintf("GS_PARENT_PID=%d", os.Getpid()))

					if err := syscall.Exec(runArgs[0], runArgs, env); err != nil {
						run_manager.Clean()
						x.Log.Fatalf("Unexpected failure: %s", err.Error())
					}
				}
				x.Log.Printf("Node uuid is %s", cs.UUID32)
			},

			// post-init stage
			func(cs *corestate.CoreState, x *app.AppX) {
				cs.Stage = corestate.StagePostInit
				x.Log.SetPrefix(logs.SetYellow(fmt.Sprintf("(%s) ", cs.Stage)))

				cs.RunDir = run_manager.Toggle()
				exist, err := utils.ExistsMatchingDirs(filepath.Join(os.TempDir(), fmt.Sprintf("/*-%s-%s", cs.UUID32, "gosally-runtime")), cs.RunDir)
				if err != nil {
					run_manager.Clean()
					x.Log.Fatalf("Unexpected failure: %s", err.Error())
				}
				if exist {
					run_manager.Clean()
					x.Log.Fatalf("Unable to continue node operation: A node with the same identifier was found in the runtime environment")
				}

				if err := run_manager.Set("run.lock"); err != nil {
					run_manager.Clean()
					x.Log.Fatalf("Unexpected failure: %s", err.Error())
				}
				lockPath, err := run_manager.Get("run.lock")
				if err != nil {
					run_manager.Clean()
					x.Log.Fatalf("Unexpected failure: %s", err.Error())
				}
				lockFile := ini.Empty()
				secRun, err := lockFile.NewSection("runtime")
				if err != nil {
					run_manager.Clean()
					x.Log.Fatalf("Unexpected failure: %s", err.Error())
				}
				secRun.Key("pid").SetValue(fmt.Sprintf("%d/%d", os.Getpid(), x.Config.Env.ParentStagePID))
				secRun.Key("version").SetValue(cs.NodeVersion)
				secRun.Key("uuid").SetValue(cs.UUID32)
				secRun.Key("timestamp").SetValue(time.Unix(cs.StartTimestampUnix, 0).Format("2006-01-02/15:04:05 MST"))
				secRun.Key("timestamp-unix").SetValue(fmt.Sprintf("%d", cs.StartTimestampUnix))

				err = lockFile.SaveTo(lockPath)
				if err != nil {
					run_manager.Clean()
					x.Log.Fatalf("Unexpected failure: %s", err.Error())
				}
			},

			func(cs *corestate.CoreState, x *app.AppX) {
				cs.Stage = corestate.StageReady
				x.Log.SetPrefix(logs.SetGreen(fmt.Sprintf("(%s) ", cs.Stage)))

				x.SLog = new(slog.Logger)
				*x.SLog = *logs.SetupLogger(x.Config.Conf.Mode)
			},
		)

		nodeApp.Run(func(ctx context.Context, cs *corestate.CoreState, x *app.AppX) error {
			ctxMain, cancelMain := context.WithCancel(ctx)
			runLockFile := run_manager.File("run.lock")
			_, err := runLockFile.Open()
			if err != nil {
				x.Log.Fatalf("cannot open run.lock: %s", err)
			}

			_, err = runLockFile.Watch(ctxMain, func() {
				x.Log.Printf("run.lock was touched")
				run_manager.Clean()
				cancelMain()
			})
			if err != nil {
				x.Log.Printf("watch error: %s", err)
			}

			serverv1 := sv1.InitV1Server(&sv1.HandlerV1InitStruct{
				Log:            *x.SLog,
				Config:         x.Config.Conf,
				AllowedCmd:     regexp.MustCompile(`^[a-zA-Z0-9]+$`),
				ListAllowedCmd: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
				Ver:            "v1",
			})

			s := gs.InitGeneral(&gs.GeneralServerInit{
				Log:    *x.SLog,
				Config: x.Config.Conf,
			}, serverv1)

			r := chi.NewRouter()
			r.Use(cors.Handler(cors.Options{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
				AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
				AllowCredentials: true,
				MaxAge:           300,
			}))
			r.Route(config.ApiRoute+config.ComDirRoute, func(r chi.Router) {
				r.Get("/", s.HandleList)
				r.Get("/{cmd}", s.Handle)
			})
			r.Route("/favicon.ico", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				})
			})

			srv := &http.Server{
				Addr:    x.Config.Conf.HTTPServer.Address,
				Handler: r,
			}

			go func() {
				if x.Config.Conf.TLS.TlsEnabled {
					x.SLog.Info("HTTPS server started with TLS", slog.String("address", x.Config.Conf.HTTPServer.Address))
					listener, err := net.Listen("tcp", x.Config.Conf.HTTPServer.Address)
					if err != nil {
						x.SLog.Error("Failed to start TLS listener", slog.String("error", err.Error()))
						return
					}
					limitedListener := netutil.LimitListener(listener, 100)
					if err := http.ServeTLS(limitedListener, r, x.Config.Conf.TLS.CertFile, x.Config.Conf.TLS.KeyFile); err != nil {
						x.SLog.Error("Failed to start HTTPS server", slog.String("error", err.Error()))
					}
				} else {
					x.SLog.Info("HTTP server started", slog.String("address", x.Config.Conf.HTTPServer.Address))
					listener, err := net.Listen("tcp", x.Config.Conf.HTTPServer.Address)
					if err != nil {
						x.SLog.Error("Failed to start listener", slog.String("error", err.Error()))
						return
					}
					limitedListener := netutil.LimitListener(listener, 100)
					if err := http.Serve(limitedListener, r); err != nil {
						x.SLog.Error("Failed to start HTTP server", slog.String("error", err.Error()))
					}
				}
				srv.Shutdown(ctxMain)
			}()

			if x.Config.Conf.Updates.UpdatesEnabled {
				go func() {
					x.Updated = update.NewUpdater(ctxMain, x.Log, x.Config.Conf, x.Config.Env)
					x.Updated.Shutdownfunc(cancelMain)
					for {
						isNewUpdate, err := x.Updated.CkeckUpdates()
						if err != nil {
							x.Log.Printf("Failed to check for updates: %s", err.Error())
						}
						if isNewUpdate {
							if err := x.Updated.Update(); err != nil {
								x.Log.Printf("Failed to update: %s", err.Error())
							} else {
								x.Log.Printf("Update completed successfully")
							}
						}
						time.Sleep(x.Config.Conf.Updates.CheckInterval)
					}
				}()
			}

			<-ctxMain.Done()

			x.Log.Println("cleaning up...")

			if err := run_manager.Clean(); err != nil {
				x.Log.Printf("cleanup error: %s", err)
			}
			x.Log.Println("bye!")

			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
