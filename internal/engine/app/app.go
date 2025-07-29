package app

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/akyaiy/GoSally-mvp/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/internal/engine/config"
)

type AppContract interface {
	InitialHooks(fn ...func(cs *corestate.CoreState, x *AppX))
	Run(fn func(ctx context.Context, cs *corestate.CoreState, x *AppX) error)
	Fallback(fn func(ctx context.Context, cs *corestate.CoreState, x *AppX))

	CallFallback(ctx context.Context)
}

type App struct {
	initHooks []func(cs *corestate.CoreState, x *AppX)
	runHook   func(ctx context.Context, cs *corestate.CoreState, x *AppX) error
	fallback  func(ctx context.Context, cs *corestate.CoreState, x *AppX)

	Corestate *corestate.CoreState
	AppX      *AppX

	fallbackOnce sync.Once
}

type AppX struct {
	Config *config.Compositor
	Log    *log.Logger
	SLog   *slog.Logger
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

func (a *App) Fallback(fn func(ctx context.Context, cs *corestate.CoreState, x *AppX)) {
	a.fallback = fn
}

func (a *App) Run(fn func(ctx context.Context, cs *corestate.CoreState, x *AppX) error) {
	a.runHook = fn

	for _, hook := range a.initHooks {
		hook(a.Corestate, a.AppX)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	defer func() {
		if r := recover(); r != nil {
			a.AppX.Log.Printf("PANIC recovered: %v", r)
			if a.fallback != nil {
				a.fallback(ctx, a.Corestate, a.AppX)
			}
			os.Exit(1)
		}
	}()

	var runErr error
	if a.runHook != nil {
		runErr = a.runHook(ctx, a.Corestate, a.AppX)
	}

	if runErr != nil {
		a.AppX.Log.Fatalf("fatal in Run: %v", runErr)
	}
}

func (a *App) CallFallback(ctx context.Context) {
	a.fallbackOnce.Do(func() {
		if a.fallback != nil {
			a.fallback(ctx, a.Corestate, a.AppX)
		}
	})
}
