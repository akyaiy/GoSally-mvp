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
	"regexp"
	"strings"

	"github.com/akyaiy/GoSally-mvp/core/config"
	"github.com/akyaiy/GoSally-mvp/core/run_manager"
	"gopkg.in/natefinch/lumberjack.v2"
)

var GlobalLevel slog.Level

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
func SetupLogger(o config.Log) (*slog.Logger, error) {
	var handlerOpts = slog.HandlerOptions{}
	var writer io.Writer = os.Stdout

	switch o.Level {
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

	if o.OutPath != "" {
		repl := map[string]string{
			"tmp": filepath.Clean(run_manager.RuntimeDir()),
		}
		re := regexp.MustCompile(`%(\w+)%`)
		result := re.ReplaceAllStringFunc(o.OutPath, func(match string) string {
			sub := re.FindStringSubmatch(match)
			if len(sub) < 2 {
				return match
			}
			key := sub[1]
			if val, ok := repl[key]; ok {
				return val
			}
			return match
		})

		if strings.Contains(o.OutPath, "%tmp%") {
			relPath := strings.TrimPrefix(result, filepath.Clean(run_manager.RuntimeDir()))
			if err := run_manager.SetDir(relPath); err != nil {
				return nil, err
			}
		}

		logFile := &lumberjack.Logger{
			Filename:   filepath.Join(result, "event.log"),
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
