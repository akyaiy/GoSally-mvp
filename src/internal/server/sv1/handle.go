package sv1

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/akyaiy/GoSally-mvp/src/internal/server/rpc"
)

func (h *HandlerV1) Handle(_ context.Context, sid string, r *http.Request, req *rpc.RPCRequest) *rpc.RPCResponse {
	if req.Method == "" {
		h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrMethodNotFoundS), slog.String("requested-method", req.Method))
		return rpc.NewError(rpc.ErrMethodIsMissing, rpc.ErrMethodIsMissingS, nil, req.ID)
	}

	method, err := h.resolveMethodPath(req.Method)
	if err != nil {
		if err.Error() == rpc.ErrInvalidMethodFormatS {
			h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrInvalidMethodFormatS), slog.String("requested-method", req.Method))
			return rpc.NewError(rpc.ErrInvalidMethodFormat, rpc.ErrInvalidMethodFormatS, nil, req.ID)
		} else if err.Error() == rpc.ErrMethodNotFoundS {
			h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrMethodNotFoundS), slog.String("requested-method", req.Method))
			return rpc.NewError(rpc.ErrMethodNotFound, rpc.ErrMethodNotFoundS, nil, req.ID)
		}
	}
	switch req.Params.(type) {
	case map[string]any, []any, nil:
		return h.handleLUA(sid, r, req, method)
	default:
		// JSON-RPC 2.0 Specification:
		// https://www.jsonrpc.org/specification#parameter_structures
		//
		// "params" MUST be either an *array* or an *object* if included.
		// Any other type (e.g., a number, string, or boolean) is INVALID.
		h.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrInvalidParamsS))
		return rpc.NewError(rpc.ErrInvalidParams, rpc.ErrInvalidParamsS, nil, req.ID)
	}
}
