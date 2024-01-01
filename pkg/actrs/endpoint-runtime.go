package actrs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/anthdm/ffaas/pkg/storage"
	"github.com/anthdm/ffaas/pkg/types"
	"github.com/anthdm/ffaas/proto"
	"github.com/anthdm/hollywood/actor"
	"github.com/google/uuid"
	"github.com/stealthrocket/wasi-go"
	"github.com/stealthrocket/wasi-go/imports"
	"github.com/tetratelabs/wazero"
	wapi "github.com/tetratelabs/wazero/api"
	"github.com/vmihailenco/msgpack/v5"
)

const KindEndpointRuntime = "endpoint_runtime"

// EndpointRuntime is an actor that can execute compiled WASM blobs in a distributed cluster.
type EndpointRuntime struct {
	store storage.Store
	cache storage.ModCacher
}

func NewEndpointRuntime(store storage.Store, cache storage.ModCacher) actor.Producer {
	return func() actor.Receiver {
		return &EndpointRuntime{
			store: store,
			cache: cache,
		}
	}
}

func (r *EndpointRuntime) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
	case actor.Stopped:
	case *proto.HTTPRequest:
		e, err := r.store.GetApp(uuid.MustParse(msg.EndpointID))
		if err != nil {
			slog.Warn("runtime could not find endpoint from store", "err", err)
			return
		}

		endpoint, ok := e.(*types.Endpoint)
		if !ok {
			slog.Warn(fmt.Sprintf("runtime could not cast endpoint type (%T)", e), "err", err)
		}

		d, err := r.store.GetDeploy(endpoint.ActiveDeployID)
		if err != nil {
			slog.Warn("runtime could not find the endpoint's active deploy from store", "err", err)
			return
		}

		deploy, ok := d.(*types.EndpointDeploy)
		if !ok {
			slog.Warn(fmt.Sprintf("runtime could not cast deploy type (%T)", d), "err", err)
		}

		httpmod, _ := NewEndpointRequestModule(msg)
		modcache, ok := r.cache.Get(endpoint.ID)
		if !ok {
			modcache = wazero.NewCompilationCache()
			slog.Warn("no cache hit", "endpoint", endpoint.ID)
		}
		r.exec(context.TODO(), deploy.Blob, modcache, endpoint.Environment, httpmod)
		resp := &proto.HTTPResponse{
			Response:   httpmod.responseBytes,
			RequestID:  msg.ID,
			StatusCode: http.StatusOK,
		}
		c.Respond(resp)
		c.Engine().Poison(c.PID())
		r.cache.Put(endpoint.ID, modcache)
	}
}

func (r *EndpointRuntime) exec(ctx context.Context, blob []byte, cache wazero.CompilationCache, env map[string]string, httpmod *EndpointRequestModule) {
	config := wazero.NewRuntimeConfig().WithCompilationCache(cache)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	defer runtime.Close(ctx)

	mod, err := runtime.CompileModule(ctx, blob)
	if err != nil {
		slog.Warn("compiling module failed", "err", err)
		return
	}
	fd := -1
	builder := imports.NewBuilder().
		WithName("ffaas").
		WithArgs().
		WithStdio(fd, fd, fd).
		WithEnv(envMapToSlice(env)...).
		// TODO: we want to mount this to some virtual folder?
		WithDirs("/").
		WithListens().
		WithDials().
		WithNonBlockingStdio(false).
		WithSocketsExtension("auto", mod).
		WithMaxOpenFiles(10).
		WithMaxOpenDirs(10)

	var system wasi.System
	ctx, system, err = builder.Instantiate(ctx, runtime)
	if err != nil {
		slog.Warn("failed to instanciate wasi module", "err", err)
		return
	}
	defer system.Close(ctx)

	httpmod.Instanciate(ctx, runtime)

	_, err = runtime.InstantiateModule(ctx, mod, wazero.NewModuleConfig())
	if err != nil {
		slog.Warn("failed to instanciate guest module", "err", err)
	}
}

func envMapToSlice(env map[string]string) []string {
	slice := make([]string, len(env))
	i := 0
	for k, v := range env {
		s := fmt.Sprintf("%s=%s", k, v)
		slice[i] = s
		i++
	}
	return slice
}

type EndpointRequestModule struct {
	requestBytes  []byte
	responseBytes []byte
}

func NewEndpointRequestModule(req *proto.HTTPRequest) (*EndpointRequestModule, error) {
	b, err := msgpack.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &EndpointRequestModule{
		requestBytes: b,
	}, nil
}

func (r *EndpointRequestModule) WriteResponse(w io.Writer) (int, error) {
	return w.Write(r.responseBytes)
}

func (r *EndpointRequestModule) Instanciate(ctx context.Context, runtime wazero.Runtime) error {
	_, err := runtime.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithGoModuleFunction(r.moduleMalloc(), []wapi.ValueType{}, []wapi.ValueType{wapi.ValueTypeI32}).
		Export("malloc").
		NewFunctionBuilder().
		WithGoModuleFunction(r.moduleWriteRequest(), []wapi.ValueType{wapi.ValueTypeI32}, []wapi.ValueType{}).
		Export("write_request").
		NewFunctionBuilder().
		WithGoModuleFunction(r.moduleWriteResponse(), []wapi.ValueType{wapi.ValueTypeI32, wapi.ValueTypeI32}, []wapi.ValueType{}).
		Export("write_response").
		Instantiate(ctx)
	return err
}

func (r *EndpointRequestModule) Close(ctx context.Context) error {
	r.responseBytes = nil
	r.requestBytes = nil
	return nil
}

func (r *EndpointRequestModule) moduleMalloc() wapi.GoModuleFunc {
	return func(ctx context.Context, module wapi.Module, stack []uint64) {
		size := uint64(len(r.requestBytes))
		stack[0] = uint64(wapi.DecodeU32(size))
	}
}

func (r *EndpointRequestModule) moduleWriteRequest() wapi.GoModuleFunc {
	return func(ctx context.Context, module wapi.Module, stack []uint64) {
		offset := wapi.DecodeU32(stack[0])
		module.Memory().Write(offset, r.requestBytes)
	}
}

func (r *EndpointRequestModule) moduleWriteResponse() wapi.GoModuleFunc {
	return func(ctx context.Context, module wapi.Module, stack []uint64) {
		offset := wapi.DecodeU32(stack[0])
		size := wapi.DecodeU32(stack[1])
		resp, _ := module.Memory().Read(offset, size)
		r.responseBytes = resp
	}
}
