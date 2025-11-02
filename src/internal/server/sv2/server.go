// SV2 works with binaries, scripts, and anything else that has access to stdin/stdout.
// Modules run in a separate process and communicate via I/O.
package sv2

import (
	"regexp"

	"github.com/akyaiy/GoSally-mvp/src/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/src/internal/engine/app"
)

// HandlerV2InitStruct structure is only for initialization
type HandlerInitStruct struct {
	Ver        string
	CS         *corestate.CoreState
	X          *app.AppX
	AllowedCmd *regexp.Regexp
}

type Handler struct {
	cs *corestate.CoreState
	x  *app.AppX

	// allowedCmd and listAllowedCmd are regular expressions used to validate command names.
	allowedCmd *regexp.Regexp

	ver string
}

func InitServer(o *HandlerInitStruct) *Handler {
	return &Handler{
		cs:         o.CS,
		x:          o.X,
		allowedCmd: o.AllowedCmd,
		ver:        o.Ver,
	}
}

// GetVersion returns the API version of the HandlerV1, which is set during initialization.
// This version is used to identify the API version in the request routing.
func (h *Handler) GetVersion() string {
	return h.ver
}
