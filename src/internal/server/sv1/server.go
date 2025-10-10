// Package sv1 provides the implementation of the Server V1 API handler.
// It includes utilities for handling API requests, extracting descriptions, and managing UUIDs.
package sv1

import (
	"regexp"

	"github.com/akyaiy/GoSally-mvp/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/internal/engine/app"
)

// HandlerV1InitStruct structure is only for initialization
type HandlerV1InitStruct struct {
	Ver        string
	CS         *corestate.CoreState
	X          *app.AppX
	AllowedCmd *regexp.Regexp
}

// HandlerV1 implements the ServerV1UtilsContract and serves as the main handler for API requests.
type HandlerV1 struct {
	cs *corestate.CoreState
	x  *app.AppX

	// allowedCmd and listAllowedCmd are regular expressions used to validate command names.
	allowedCmd *regexp.Regexp

	ver string
}

// InitV1Server initializes a new HandlerV1 with the provided configuration and returns it.
// Should be carefull with giving to this function invalid parameters,
// because there is no validation of parameters in this function.
func InitV1Server(o *HandlerV1InitStruct) *HandlerV1 {
	return &HandlerV1{
		cs:         o.CS,
		x:          o.X,
		allowedCmd: o.AllowedCmd,
		ver:        o.Ver,
	}
}

// GetVersion returns the API version of the HandlerV1, which is set during initialization.
// This version is used to identify the API version in the request routing.
func (h *HandlerV1) GetVersion() string {
	return h.ver
}
