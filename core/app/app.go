package app

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/akyaiy/GoSally-mvp/core/config"
	"github.com/akyaiy/GoSally-mvp/core/corestate"
	"github.com/akyaiy/GoSally-mvp/core/update"
)

type AppContract interface {
	InitialHooks(fn ...func(cs *corestate.CoreState, x *AppX))
	Run(fn func(ctx context.Context, cs *corestate.CoreState, x *AppX) error)
}

type App struct {
	initHooks []func(cs *corestate.CoreState, x *AppX)
	runHook   func(ctx context.Context, cs *corestate.CoreState, x *AppX) error

	Corestate *corestate.CoreState
	AppX      *AppX
}

type AppX struct {
	Config  *config.Compositor
	Log     *log.Logger
	SLog    *slog.Logger
	Updated *update.Updater
}

func New() AppContract {
	return &App{
		AppX: &AppX{
			Log: log.Default(),
		},
		Corestate: &corestate.CoreState{},
	}
}

func (a *App) InitialHooks(fn ...func(cs *corestate.CoreState, x *AppX)) {
	a.initHooks = append(a.initHooks, fn...)
}

func (a *App) Run(fn func(ctx context.Context, cs *corestate.CoreState, x *AppX) error) {
	a.runHook = fn

	for _, hook := range a.initHooks {
		hook(a.Corestate, a.AppX)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	if a.runHook != nil {
		if err := a.runHook(ctx, a.Corestate, a.AppX); err != nil {
			log.Fatalf("fatal in Run: %v", err)
		}
	}
}
