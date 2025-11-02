package sv2

import (
	"context"
	"net/http"

	"github.com/akyaiy/GoSally-mvp/src/internal/server/rpc"
)

func (h *Handler) Handle(_ context.Context, sid string, r *http.Request, req *rpc.RPCRequest) *rpc.RPCResponse {
	return nil
}
