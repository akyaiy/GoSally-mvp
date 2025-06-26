package sv1

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/akyaiy/GoSally-mvp/core/config"
)

type ServerV1UtilsContract interface {
	extractDescriptionStatic(path string) (string, error)
	writeJSONError(status int, msg string)
	newUUID() string

	_errNotFound()
	ErrNotFound(w http.ResponseWriter, r *http.Request)
}

// structure only for initialization
type HandlerV1InitStruct struct {
	Ver            string
	Log            slog.Logger
	Config         *config.ConfigConf
	AllowedCmd     *regexp.Regexp
	ListAllowedCmd *regexp.Regexp
}

type HandlerV1 struct {
	w http.ResponseWriter
	r *http.Request

	log slog.Logger

	cfg *config.ConfigConf

	allowedCmd     *regexp.Regexp
	listAllowedCmd *regexp.Regexp

	ver string
}

func InitV1Server(o *HandlerV1InitStruct) *HandlerV1 {
	return &HandlerV1{
		log:            o.Log,
		cfg:            o.Config,
		allowedCmd:     o.AllowedCmd,
		listAllowedCmd: o.ListAllowedCmd,
		ver:            o.Ver,
	}
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

func (h *HandlerV1) GetVersion() string {
	return h.ver
}
