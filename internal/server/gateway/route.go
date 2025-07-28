package gateway

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
)

func (gs *GatewayServer) Handle(w http.ResponseWriter, r *http.Request) {
	var req rpc.RPCRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		rpc.WriteRouterError(w, http.StatusBadRequest, &rpc.RPCError{
			JSONRPC: rpc.JSONRPCVersion,
			ID:      nil,
			Error: map[string]any{
				"code":    rpc.ErrInternalError,
				"message": rpc.ErrInternalErrorS,
			},
		})
		gs.log.Info("invalid request received", slog.String("issue", rpc.ErrInternalErrorS))
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		rpc.WriteRouterError(w, http.StatusBadRequest, &rpc.RPCError{
			JSONRPC: rpc.JSONRPCVersion,
			ID:      nil,
			Error: map[string]any{
				"code":    rpc.ErrParseError,
				"message": rpc.ErrParseErrorS,
			},
		})
		gs.log.Info("invalid request received", slog.String("issue", rpc.ErrParseErrorS))
		return
	}

	if req.JSONRPC != rpc.JSONRPCVersion {
		rpc.WriteRouterError(w, http.StatusBadRequest, &rpc.RPCError{
			JSONRPC: rpc.JSONRPCVersion,
			ID:      req.ID,
			Error: map[string]any{
				"code":    rpc.ErrInvalidRequest,
				"message": rpc.ErrInvalidRequestS,
			},
		})
		gs.log.Info("invalid request received", slog.String("issue", rpc.ErrInvalidRequestS), slog.String("requested-version", req.JSONRPC))
		return
	}

	gs.Route(w, r, req)
}

func (gs *GatewayServer) Route(w http.ResponseWriter, r *http.Request, req rpc.RPCRequest) {
	server, ok := gs.servers[serversApiVer(req.Params.ContextVersion)]
	if !ok {
		rpc.WriteRouterError(w, http.StatusBadRequest, &rpc.RPCError{
			JSONRPC: rpc.JSONRPCVersion,
			ID:      req.ID,
			Error: map[string]any{
				"code":    rpc.ErrContextVersion,
				"message": rpc.ErrContextVersionS,
			},
		})
		gs.log.Info("invalid request received", slog.String("issue", rpc.ErrContextVersionS), slog.String("requested-version", req.Params.ContextVersion))
		return
	}

	// checks if request is notification
	if req.ID == nil {
		rr := httptest.NewRecorder()
		server.Handle(rr, r, req)
		return
	}
	server.Handle(w, r, req)
}
