package gateway

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/akyaiy/GoSally-mvp/internal/core/utils"
	"github.com/akyaiy/GoSally-mvp/internal/server/rpc"
	"github.com/google/uuid"
)

func (gs *GatewayServer) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // TODO

	w.Header().Set("Content-Type", "application/json")
	sessionUUID := r.Header.Get("X-Session-UUID")
	if sessionUUID == "" {
		sessionUUID = uuid.New().String()

	}
	gs.x.SLog.Debug("new request", slog.String("session-uuid", sessionUUID), slog.Group("connection", slog.String("ip", r.RemoteAddr)))

	w.Header().Set("X-Session-UUID", sessionUUID)
	if !gs.sm.Add(sessionUUID) {
		gs.x.SLog.Debug("session is busy", slog.String("session-uuid", sessionUUID))
		rpc.WriteError(w, rpc.NewError(rpc.ErrSessionIsBusy, rpc.ErrSessionIsBusyS, nil, nil))
		return
	}
	defer gs.sm.Delete(sessionUUID)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		gs.x.SLog.Debug("failed to read body", slog.String("err", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		rpc.WriteError(w, rpc.NewError(rpc.ErrInternalError, rpc.ErrInternalErrorS, nil, nil))
		gs.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrInternalErrorS))
		return
	}

	// determine if the JSON-RPC request is a batch
	var batch []rpc.RPCRequest
	json.Unmarshal(body, &batch)
	var single rpc.RPCRequest
	if batch == nil {
		if err := json.Unmarshal(body, &single); err != nil {
			gs.x.SLog.Debug("failed to parse json", slog.String("err", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			rpc.WriteError(w, rpc.NewError(rpc.ErrParseError, rpc.ErrParseErrorS, nil, nil))
			gs.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrParseErrorS))
			return
		}
		resp := gs.Route(ctx, sessionUUID, r, &single)
		if resp == nil {
			w.Write([]byte(""))
			return
		}
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
			res := gs.Route(ctx, sessionUUID, r, &req)
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
	} else {
		w.Write([]byte("[]"))
	}
}

func (gs *GatewayServer) Route(ctx context.Context, sid string, r *http.Request, req *rpc.RPCRequest) (resp *rpc.RPCResponse) {
	defer utils.CatchPanicWithFallback(func(rec any) {
		gs.x.SLog.Error("panic caught in handler", slog.Any("error", rec))
		resp = rpc.NewError(rpc.ErrInternalError, "Internal server error (panic)", nil, req.ID)
	})
	if req.JSONRPC != rpc.JSONRPCVersion {
		gs.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrInvalidRequestS), slog.String("requested-version", req.JSONRPC))
		return rpc.NewError(rpc.ErrInvalidRequest, rpc.ErrInvalidRequestS, nil, req.ID)
	}

	server, ok := gs.servers[serversApiVer(req.ContextVersion)]
	if !ok {
		gs.x.SLog.Info("invalid request received", slog.String("issue", rpc.ErrContextVersionS), slog.String("requested-version", req.ContextVersion))
		return rpc.NewError(rpc.ErrContextVersion, rpc.ErrContextVersionS, nil, req.ID)
	}

	// checks if request is notification
	if req.ID == nil {
		go server.Handle(ctx, sid, r, req)
		return nil
	}
	return server.Handle(ctx, sid, r, req)
}
