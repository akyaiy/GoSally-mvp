package sv1

import (
	"log/slog"
	"net/http"

	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
)

func (h *HandlerV1) Handle(r *http.Request, req *rpc.RPCRequest) *rpc.RPCResponse {
	if req.Method == "" {
		h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrMethodNotFoundS), slog.String("requested-method", req.Method))
		return rpc.NewError(rpc.ErrMethodIsMissing, rpc.ErrMethodIsMissingS, req.ID)
	}

	method, err := h.resolveMethodPath(req.Method)
	if err != nil {
		if err.Error() == rpc.ErrInvalidMethodFormatS {
			h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrInvalidMethodFormatS), slog.String("requested-method", req.Method))
			return rpc.NewError(rpc.ErrInvalidMethodFormat, rpc.ErrInvalidMethodFormatS, req.ID)
		} else if err.Error() == rpc.ErrMethodNotFoundS {
			h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrMethodNotFoundS), slog.String("requested-method", req.Method))
			return rpc.NewError(rpc.ErrMethodNotFound, rpc.ErrMethodNotFoundS, req.ID)
		}
	}

	return h.HandleLUA(method, req)
}
