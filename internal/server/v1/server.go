package server_v1

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/akyaiy/GoSally-mvp/internal/config"
)

type ServerV1UtilsContract interface {
	extractDescriptionStatic(path string) (string, error)
	writeJSONError(status int, msg string)
	newUUID() string
	errNotFound()
}

type ServerV1Contract interface {
	ServerV1UtilsContract

	Handle(w http.ResponseWriter, r *http.Request)
	HandleList(w http.ResponseWriter, r *http.Request)

	_handle()
	_handleList()
}

type HandlerV1 struct {
	w http.ResponseWriter
	r *http.Request

	log slog.Logger

	cfg *config.ConfigConf

	allowedCmd     *regexp.Regexp
	listAllowedCmd *regexp.Regexp
}

func (h *HandlerV1) Handle(w http.ResponseWriter, r *http.Request) {
	h.w = w
	h.r = r
	h._handle()
}

func (h *HandlerV1) HandleList(w http.ResponseWriter, r *http.Request) {
	h.w = w
	h.r = r
	h._handleList()
}
