package gateway

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/akyaiy/GoSally-mvp/internal/core/utils"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
)

func (gs *GatewayServer) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		rpc.WriteError(w, &rpc.RPCResponse{
			JSONRPC: rpc.JSONRPCVersion,
			ID:      nil,
			Error: map[string]any{
				"code":    rpc.ErrInternalError,
				"message": rpc.ErrInternalErrorS,
			},
		})
		gs.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrInternalErrorS))
		return
	}

	// determine if the JSON-RPC request is a batch
	var batch []rpc.RPCRequest
	json.Unmarshal(body, &batch)
	var single rpc.RPCRequest
	if batch == nil {
		if err := json.Unmarshal(body, &single); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			rpc.WriteError(w, &rpc.RPCResponse{
				JSONRPC: rpc.JSONRPCVersion,
				ID:      nil,
				Error: map[string]any{
					"code":    rpc.ErrParseError,
					"message": rpc.ErrParseErrorS,
				},
			})
			gs.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrParseErrorS))
			return
		}
		resp := gs.Route(r, &single)
		rpc.WriteResponse(w, resp)
		return
	}

	// handle batch
	responses := make(chan rpc.RPCResponse, len(batch))
	var wg sync.WaitGroup
	for _, m := range batch {
		wg.Add(1)
		go func(req rpc.RPCRequest) {
			defer wg.Done()
			res := gs.Route(r, &req)
			if res != nil {
				responses <- *res
			}
		}(m)
	}
	wg.Wait()
	close(responses)

	var result []rpc.RPCResponse
	for res := range responses {
		result = append(result, res)
	}
	if len(result) > 0 {
		json.NewEncoder(w).Encode(result)
	}
}

func (gs *GatewayServer) Route(r *http.Request, req *rpc.RPCRequest) (resp *rpc.RPCResponse) {
	defer utils.CatchPanicWithFallback(func(rec any) {
		gs.x.SLog.Error("panic caught in handler", slog.Any("error", rec))
		resp = rpc.NewError(rpc.ErrInternalError, "Internal server error (panic)", req.ID)
	})
	if req.JSONRPC != rpc.JSONRPCVersion {
		gs.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrInvalidRequestS), slog.String("requested-version", req.JSONRPC))
		return rpc.NewError(rpc.ErrInvalidRequest, rpc.ErrInvalidRequestS, req.ID)
	}

	server, ok := gs.servers[serversApiVer(req.ContextVersion)]
	if !ok {
		gs.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrContextVersionS), slog.String("requested-version", req.ContextVersion))
		return rpc.NewError(rpc.ErrContextVersion, rpc.ErrContextVersionS, req.ID)
	}

	resp = server.Handle(r, req)
	// checks if request is notification
	if req.ID == nil {
		return nil
	}
	return resp
}
