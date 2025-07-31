package sv1

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
)

func (h *HandlerV1) resolveMethodPath(method string) (string, error) {
	if !h.allowedCmd.MatchString(method) {
		return "", errors.New(rpc.ErrInvalidMethodFormatS)
	}

	parts := strings.Split(method, ">")
	relPath := filepath.Join(parts...) + ".lua"
	fullPath := filepath.Join(*h.x.Config.Conf.Node.ComDir, relPath)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", errors.New(rpc.ErrMethodNotFoundS)
	}

	return fullPath, nil
}
