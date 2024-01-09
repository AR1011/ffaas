package actrs

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/raptor/internal/runtime"
	"github.com/anthdm/raptor/internal/shared"
	"github.com/anthdm/raptor/internal/spidermonkey"
	"github.com/anthdm/raptor/internal/storage"
	"github.com/anthdm/raptor/internal/types"
	"github.com/anthdm/raptor/proto"
	"github.com/google/uuid"
	"github.com/tetratelabs/wazero"

	prot "google.golang.org/protobuf/proto"
)

const KindRuntime = "runtime"

// Runtime is an actor that can execute compiled WASM blobs in a distributed cluster.
type Runtime struct {
	store    storage.Store
	cache    storage.ModCacher
	started  time.Time
	deployID uuid.UUID
}

func NewRuntime(store storage.Store, cache storage.ModCacher) actor.Producer {
	return func() actor.Receiver {
		return &Runtime{
			store: store,
			cache: cache,
		}
	}
}

func (r *Runtime) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		r.started = time.Now()
	case actor.Stopped:
	case *proto.HTTPRequest:
		// Handle the HTTP request that is forwarded from the WASM server actor.
		r.handleHTTPRequest(c, msg)
	}
}

func (r *Runtime) handleHTTPRequest(ctx *actor.Context, msg *proto.HTTPRequest) {
	r.deployID = uuid.MustParse(msg.DeploymentID)
	st := time.Now()
	deploy, err := r.store.GetDeployment(r.deployID)
	if err != nil {
		slog.Warn("runtime could not find deploy from store", "err", err, "id", r.deployID)
		respondError(ctx, http.StatusInternalServerError, "internal server error", msg.ID)
		return
	}
	fmt.Printf("get deployment : %s\n", time.Since(st))

	st = time.Now()
	modCache, ok := r.cache.Get(deploy.EndpointID)
	if !ok {
		modCache = wazero.NewCompilationCache()
		slog.Warn("no cache hit", "endpoint", deploy.EndpointID)
	}
	fmt.Printf("get cache : %s\n", time.Since(st))

	st = time.Now()
	b, err := prot.Marshal(msg)
	if err != nil {
		slog.Warn("failed to marshal incoming HTTP request", "err", err)
		respondError(ctx, http.StatusInternalServerError, "internal server error", msg.ID)
		return
	}
	fmt.Printf("marshal : %s\n", time.Since(st))

	st = time.Now()
	in := bytes.NewReader(b)
	out := &bytes.Buffer{}
	args := runtime.InvokeArgs{
		Env:   msg.Env,
		In:    in,
		Out:   out,
		Cache: modCache,
		Debug: true,
	}
	fmt.Printf("args : %s\n", time.Since(st))

	switch msg.Runtime {
	case "go":
		args.Blob = deploy.Blob
	case "js":
		args.Blob = spidermonkey.WasmBlob
		args.Args = []string{"", "-e", string(deploy.Blob)}
	default:
		err = fmt.Errorf("invalid runtime: %s", msg.Runtime)
	}

	st = time.Now()
	err = runtime.Invoke(context.Background(), args)
	if err != nil {
		slog.Error("runtime invoke error", "err", err)
		respondError(ctx, http.StatusInternalServerError, "internal server error", msg.ID)
		return
	}
	fmt.Printf("invoke : %s\n", time.Since(st))

	st = time.Now()
	res, status, err := shared.ParseRuntimeHTTPResponse(out.String())
	if err != nil {
		respondError(ctx, http.StatusInternalServerError, "internal server error", msg.ID)
		return
	}
	fmt.Printf("parse response : %s\n", time.Since(st))
	resp := &proto.HTTPResponse{
		Response:   []byte(res),
		RequestID:  msg.ID,
		StatusCode: int32(status),
	}

	st = time.Now()
	ctx.Respond(resp)
	fmt.Printf("respond : %s\n", time.Since(st))

	st = time.Now()
	r.cache.Put(deploy.EndpointID, modCache)
	fmt.Printf("put cache : %s\n", time.Since(st))

	st = time.Now()
	ctx.Engine().Poison(ctx.PID())
	fmt.Printf("poison : %s\n", time.Since(st))

	// only send metrics when its a request on LIVE
	if !msg.Preview {
		metric := types.RuntimeMetric{
			ID:           uuid.New(),
			StartTime:    r.started,
			Duration:     time.Since(r.started),
			DeploymentID: deploy.ID,
			EndpointID:   deploy.EndpointID,
			RequestURL:   msg.URL,
			StatusCode:   status,
		}
		pid := ctx.Engine().Registry.GetPID(KindMetric, "1")
		ctx.Send(pid, metric)
	}
}

func respondError(ctx *actor.Context, code int32, msg string, id string) {
	ctx.Respond(&proto.HTTPResponse{
		Response:   []byte(msg),
		StatusCode: code,
		RequestID:  id,
	})
}
