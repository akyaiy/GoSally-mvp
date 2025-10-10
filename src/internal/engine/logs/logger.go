// Package logs provides a logger setup function that configures the logger based on the environment.
// It supports different logging levels for development and production environments.
// It uses the standard library's slog package for structured logging.
package logs

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/akyaiy/GoSally-mvp/internal/engine/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

var GlobalLevel slog.Level

type levelsStruct struct {
	Available []string
	Fallback  string
}

var Levels = levelsStruct{
	Available: []string{
		"debug", "info",
	},
	Fallback: "info",
}

type SlogWriter struct {
	Logger *slog.Logger
	Level  slog.Level
}

func (w *SlogWriter) Write(p []byte) (n int, err error) {
	msg := string(bytes.TrimSpace(p))
	w.Logger.Log(context.TODO(), w.Level, msg)
	return len(p), nil
}

// SetupLogger initializes and returns a logger based on the provided environment.
func SetupLogger(o *config.Log) (*slog.Logger, error) {
	var handlerOpts = slog.HandlerOptions{}
	var writer io.Writer = os.Stdout

	switch *o.Level {
	case "debug":
		GlobalLevel = slog.LevelDebug
		handlerOpts.Level = slog.LevelDebug
	case "info":
		GlobalLevel = slog.LevelInfo
		handlerOpts.Level = slog.LevelInfo
	default:
		GlobalLevel = slog.LevelInfo
		handlerOpts.Level = slog.LevelInfo
	}

	switch *o.OutPath {
	case "_1STDout":
		writer = os.Stdout
	case "_2STDerr":
		writer = os.Stderr
	default:
		logFile := &lumberjack.Logger{
			Filename:   filepath.Join(*o.OutPath, "event.log"),
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     28,
			Compress:   true,
		}
		writer = logFile
	}

	var handler slog.Handler

	if *o.JSON {
		handler = slog.NewJSONHandler(writer, &handlerOpts)
	} else {
		handler = slog.NewTextHandler(writer, &handlerOpts)
	}
	log := slog.New(handler)
	return log, nil
}
