package actrs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthdm/ffaas/pkg/storage"
	"github.com/anthdm/ffaas/pkg/types"
	"github.com/anthdm/ffaas/proto"
	"github.com/anthdm/hollywood/actor"
	"github.com/google/uuid"
	"github.com/stealthrocket/wasi-go"
	"github.com/stealthrocket/wasi-go/imports"
	"github.com/tetratelabs/wazero"
)

const KindCronRuntime = "cron_runtime"

// CronRuntime is an actor that can execute compiled WASM blobs in a distributed cluster.
type CronRuntime struct {
	store storage.Store
	cache storage.ModCacher
	cron  *types.Cron
}

// message to start the execution of the blob
type RunCron struct{}

func NewCronRuntime(store storage.Store, cache storage.ModCacher) actor.Producer {
	return func() actor.Receiver {
		return &CronRuntime{
			store: store,
			cache: cache,
		}
	}
}

func (r *CronRuntime) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
	case actor.Stopped:

	case *proto.StartCronJob:
		r.StartCron(uuid.MustParse(msg.ID), c)

	case *proto.StopCronJob:
		c.Engine().Poison(c.PID())

	case RunCron:
		d, err := r.store.GetDeploy(r.cron.ActiveDeployID)
		if err != nil {
			slog.Warn("runtime could not find the crons's active deploy from store", "err", err)
			return
		}

		deploy, ok := d.(*types.CronDeploy)
		if !ok {
			slog.Warn(fmt.Sprintf("runtime could not cast deploy type (%T)", d), "err", err)
		}

		modcache, ok := r.cache.Get(r.cron.ID)
		if !ok {
			modcache = wazero.NewCompilationCache()
			slog.Warn("no cache hit", "cron", r.cron.ID)
		}
		r.exec(context.TODO(), deploy.Blob, modcache, r.cron.Environment)
		r.cache.Put(r.cron.ID, modcache)
	}
}

func (r *CronRuntime) StartCron(id uuid.UUID, ctx *actor.Context) {
	cron, err := r.store.GetApp(id)
	if err != nil {
		slog.Warn("runtime could not find cron from store", "err", err)
		return
	}

	c, ok := cron.(*types.Cron)
	if !ok {
		slog.Warn(fmt.Sprintf("runtime could not cast cron type (%T)", cron), "err", err)
		return
	}
	r.cron = c

	ctx.Engine().SendRepeat(ctx.PID(), RunCron{}, time.Duration(c.Interval)*time.Second)

}

func (r *CronRuntime) exec(ctx context.Context, blob []byte, cache wazero.CompilationCache, env map[string]string) {
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

	_, err = runtime.InstantiateModule(ctx, mod, wazero.NewModuleConfig())
	if err != nil {
		slog.Warn("failed to instanciate guest module", "err", err)
	}
}
