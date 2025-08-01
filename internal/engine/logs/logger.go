// Package logs provides a logger setup function that configures the logger based on the environment.
// It supports different logging levels for development and production environments.
// It uses the standard library's slog package for structured logging.
package logs

import (
	"bytes"
	"context"
	"fmt"
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

	switch  o.OutPath{
	case 1:
		writer = os.Stdout
	case 2:
		writer = os.Stderr
	case os.Stdout:
		writer = os.Stdout
	case os.Stderr:
		writer = os.Stderr
	default:
		var path string
		switch v := o.OutPath.(type) {
		case string:
			path = v
		case int, int64, float64:
			path = fmt.Sprint(v)
		case fmt.Stringer:
			path = v.String()
		default:
			path = fmt.Sprint(v)
		}

		logFile := &lumberjack.Logger{
			Filename:   filepath.Join(path, "event.log"),
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     28,
			Compress:   true,
		}
		writer = logFile
	}

	log := slog.New(slog.NewJSONHandler(writer, &handlerOpts))
	return log, nil
}
