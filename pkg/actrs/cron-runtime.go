package actrs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/run/pkg/storage"
	"github.com/anthdm/run/pkg/types"
	"github.com/anthdm/run/proto"
	"github.com/google/uuid"
	"github.com/stealthrocket/wasi-go"
	"github.com/stealthrocket/wasi-go/imports"
	"github.com/tetratelabs/wazero"
)

const KindTaskRuntime = "task_runtime"

// TaskRuntime is an actor that can execute compiled WASM blobs in a distributed cluster.
type TaskRuntime struct {
	store   storage.Store
	cache   storage.ModCacher
	task    *types.Task
	started time.Time
}

// message to start the execution of the blob
type RunTask struct{}

func NewTaskRuntime(store storage.Store, cache storage.ModCacher) actor.Producer {
	return func() actor.Receiver {
		return &TaskRuntime{
			store: store,
			cache: cache,
		}
	}
}

func (r *TaskRuntime) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		r.started = time.Now()
	case actor.Stopped:

	case *proto.StartTask:
		r.StartTask(uuid.MustParse(msg.ID), c)

	case *proto.StopTask:
		c.Engine().Poison(c.PID())

	case RunTask:
		d, err := r.store.GetDeploy(r.task.ActiveDeployID)
		if err != nil {
			slog.Warn("runtime could not find the task's active deploy from store", "err", err)
			return
		}

		deploy, ok := d.(*types.TaskDeploy)
		if !ok {
			slog.Warn(fmt.Sprintf("runtime could not cast deploy type (%T)", d), "err", err)
		}

		modcache, ok := r.cache.Get(r.task.ID)
		if !ok {
			modcache = wazero.NewCompilationCache()
			slog.Warn("no cache hit", "task", r.task.ID)
		}
		r.exec(context.TODO(), deploy.Blob, modcache, r.task.Environment)
		r.cache.Put(r.task.ID, modcache)
	}
}

func (r *TaskRuntime) StartTask(id uuid.UUID, ctx *actor.Context) {
	task, err := r.store.GetApp(id)
	if err != nil {
		slog.Warn("runtime could not find task from store", "err", err)
		return
	}

	c, ok := task.(*types.Task)
	if !ok {
		slog.Warn(fmt.Sprintf("runtime could not cast task type (%T)", task), "err", err)
		return
	}
	r.task = c

	ctx.Engine().SendRepeat(ctx.PID(), RunTask{}, time.Duration(c.Interval)*time.Second)

}

func (r *TaskRuntime) exec(ctx context.Context, blob []byte, cache wazero.CompilationCache, env map[string]string) {
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
		WithName("run").
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
