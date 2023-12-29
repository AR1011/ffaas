package runtime

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/anthdm/ffaas/pkg/storage"
	"github.com/anthdm/ffaas/pkg/utils"
	"github.com/google/uuid"
	"github.com/stealthrocket/wasi-go"
	"github.com/stealthrocket/wasi-go/imports"
	"github.com/stealthrocket/wasi-go/imports/wasi_http"
	"github.com/tetratelabs/wazero"
	wapi "github.com/tetratelabs/wazero/api"
	"github.com/vmihailenco/msgpack/v5"
)

type request struct {
	Body   []byte
	Method string
	URL    string
}

type RequestPlugin interface {
	Instanciate(context.Context, wazero.Runtime) error
	WriteResponse(io.Writer) (int, error)
	Close(context.Context) error
}

type RequestModule struct {
	requestBytes  []byte
	responseBytes []byte
}

// TODO: could probably do more optimized stuff for larger bodies.
// We want to limit the body size though...
func NewRequestModule(r *http.Request) (*RequestModule, error) {
	if r == nil {
		return nil, fmt.Errorf("http.request is nil")
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	req := request{
		Body:   b,
		Method: r.Method,
		URL:    r.URL.RequestURI(),
	}

	b, err = msgpack.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &RequestModule{
		requestBytes: b,
	}, nil
}

func (r *RequestModule) WriteResponse(w io.Writer) (int, error) {
	return w.Write(r.responseBytes)
}

func (r *RequestModule) Instanciate(ctx context.Context, runtime wazero.Runtime) error {
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

func (r *RequestModule) Close(ctx context.Context) error {
	r.responseBytes = nil
	r.requestBytes = nil
	return nil
}

func (r *RequestModule) moduleMalloc() wapi.GoModuleFunc {
	return func(ctx context.Context, module wapi.Module, stack []uint64) {
		size := uint64(len(r.requestBytes))
		stack[0] = uint64(wapi.DecodeU32(size))
	}
}

func (r *RequestModule) moduleWriteRequest() wapi.GoModuleFunc {
	return func(ctx context.Context, module wapi.Module, stack []uint64) {
		offset := wapi.DecodeU32(stack[0])
		module.Memory().Write(offset, r.requestBytes)
	}
}

func (r *RequestModule) moduleWriteResponse() wapi.GoModuleFunc {
	return func(ctx context.Context, module wapi.Module, stack []uint64) {
		offset := wapi.DecodeU32(stack[0])
		size := wapi.DecodeU32(stack[1])
		resp, _ := module.Memory().Read(offset, size)
		r.responseBytes = resp
	}
}

func Compile(ctx context.Context, cache wazero.CompilationCache, blob []byte) error {
	config := wazero.NewRuntimeConfig().WithCompilationCache(cache)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	defer runtime.Close(ctx)

	_, err := runtime.CompileModule(ctx, blob)
	if err != nil {
		return err
	}
	return nil
}

type Args struct {
	Blob          []byte
	Cache         wazero.CompilationCache
	RequestPlugin RequestPlugin
	Env           map[string]string
	RequestID     uuid.UUID
	AppID         uuid.UUID

	Store storage.Store
}

func Run(ctx context.Context, args Args) error {
	config := wazero.NewRuntimeConfig().WithCompilationCache(args.Cache)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	defer runtime.Close(ctx)

	if err := args.RequestPlugin.Instanciate(ctx, runtime); err != nil {
		return err
	}

	wasmModule, err := runtime.CompileModule(ctx, args.Blob)
	if err != nil {
		return err
	}
	// TODO: Can't close cause it will invalidate the cache.
	// defer wasmModule.Close(ctx)

	// TODO: probably handled better
	tempDir, err := utils.CreateTemporyWorkDir(args.RequestID)
	if err != nil {
		log.Fatal(err)
	}

	stdout, stderr := utils.CreateTempStdioFiles(args.RequestID)

	defer func() {
		// reads the logs from temp files and closes them
		stdoutLogs, stderrLogs := utils.CleanupTempStdioFiles(stdout, stderr)

		// appends the logs to the application logs
		if args.Store != nil {
			_ = args.Store.AppendApplicationLogs(args.AppID, stdoutLogs, stderrLogs)
		} else {
			log.Println("no store provided")
		}

		// removes the temp work dir and closes files
		utils.CleanupTemporaryWorkDir(args.RequestID)
	}()

	builder := imports.NewBuilder().
		WithName("ffaas").
		WithArgs().
		WithStdio(-1, int(stdout.Fd()), int(stderr.Fd())).
		// TODO: env...
		WithEnv().
		WithDirs(tempDir).
		WithListens().
		WithDials().
		WithNonBlockingStdio(false).
		WithSocketsExtension("auto", wasmModule).
		WithMaxOpenFiles(10).
		WithMaxOpenDirs(10)

	var system wasi.System
	ctx, system, err = builder.Instantiate(ctx, runtime)
	if err != nil {
		return err
	}
	defer system.Close(ctx)

	wasiHTTP := wasi_http.MakeWasiHTTP()
	if err := wasiHTTP.Instantiate(ctx, runtime); err != nil {
		return err
	}

	_, err = runtime.InstantiateModule(ctx, wasmModule, wazero.NewModuleConfig())

	return err
}
