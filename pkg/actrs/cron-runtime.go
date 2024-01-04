package actrs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/run/pkg/spidermonkey"
	"github.com/anthdm/run/pkg/storage"
	"github.com/anthdm/run/pkg/types"
	"github.com/anthdm/run/proto"
	"github.com/google/uuid"
	"github.com/stealthrocket/wasi-go"
	"github.com/stealthrocket/wasi-go/imports"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const KindTaskRuntime = "task_runtime"

// TaskRuntime is an actor that can execute compiled WASM blobs in a distributed cluster.
type TaskRuntime struct {
	store       storage.Store
	metricStore storage.MetricStore
	cache       storage.ModCacher
	task        *types.Task
	started     time.Time
	repeater    *actor.SendRepeater
}

// message to start the execution of the blob
type RunTask struct{}

func NewTaskRuntime(store storage.Store, metrics storage.MetricStore, cache storage.ModCacher) actor.Producer {
	return func() actor.Receiver {
		return &TaskRuntime{
			store:       store,
			cache:       cache,
			metricStore: metrics,
		}
	}
}

func (r *TaskRuntime) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		r.started = time.Now()
	case actor.Stopped:

	case *proto.StartRequest:
		r.StartTask(msg, c)

	case *proto.StopRequest:
		r.StopTask(msg, c)

	case RunTask:
		d, err := r.store.GetDeploy(r.task.ActiveDeployID)
		if err != nil {
			slog.Warn("runtime could not find the task's active deploy from store", "err", err)
			return
		}

		deploy, ok := d.(*types.TaskDeploy)
		if !ok {
			slog.Warn(fmt.Sprintf("runtime could not cast deploy type (%T)", d), "err", err)
			return
		}

		modcache, ok := r.cache.Get(r.task.ID)
		if !ok {
			modcache = wazero.NewCompilationCache()
			slog.Warn("no cache hit", "task", r.task.ID)
		}
		r.cache.Put(r.task.ID, modcache)

		start := time.Now()
		switch r.task.Runtime {
		case "js":
			r.invokeJSRuntime(context.TODO(), deploy.Blob, os.Stdout, r.task.Environment)
		case "go":
			r.invokeGoRuntime(context.TODO(), deploy.Blob, modcache, r.task.Environment)
		}

		metric := types.TaskRuntimeMetric{
			ID:        uuid.New(),
			Type:      types.TaskMetricType,
			StartTime: start,
			Duration:  time.Since(start),
			DeployID:  deploy.ID,
			TaskID:    r.task.ID,
		}

		if err := r.metricStore.CreateRuntimeMetric(&metric); err != nil {
			slog.Warn("failed to create runtime metric", "err", err)
		}

	}
}

// starts the execution of the task
// uses sendRepeat with interval to repeatdily run the wasm
func (r *TaskRuntime) StartTask(msg *proto.StartRequest, c *actor.Context) {
	id := uuid.MustParse(msg.ID)
	t, err := r.store.GetApp(id)
	if err != nil {
		slog.Warn("runtime could not find task from store", "err", err)
		c.Respond(&proto.StartStopResponse{
			ID:        id.String(),
			RequestID: msg.RequestID,
			Err:       fmt.Sprintf("runtime could not find task from store (%s)", err),
		})
		return
	}

	task, ok := t.(*types.Task)
	if !ok {
		slog.Warn(fmt.Sprintf("runtime could not cast task type (%T)", task), "err", err)
		c.Respond(&proto.StartStopResponse{
			ID:        id.String(),
			RequestID: msg.RequestID,
			Err:       fmt.Sprintf("runtime could not cast task type (%T)", task),
		})
		return
	}
	r.task = task

	repeater := c.Engine().SendRepeat(c.PID(), RunTask{}, time.Duration(task.Interval)*time.Second)
	r.repeater = &repeater

	c.Respond(&proto.StartStopResponse{
		ID:        id.String(),
		RequestID: msg.RequestID,
		Err:       "",
	})
}

func (r *TaskRuntime) StopTask(msg *proto.StopRequest, c *actor.Context) {
	id := uuid.MustParse(msg.ID)
	if r.repeater == nil {
		c.Respond(&proto.StartStopResponse{
			ID:        id.String(),
			RequestID: msg.RequestID,
			Err:       fmt.Sprintf("task (%s) is not running", id),
		})
		return
	}

	r.repeater.Stop()

	c.Respond(&proto.StartStopResponse{
		ID:        id.String(),
		RequestID: msg.RequestID,
		Err:       "",
	})

	c.Engine().Poison(c.PID())
}

func (r *TaskRuntime) invokeGoRuntime(ctx context.Context,
	blob []byte,
	cache wazero.CompilationCache,
	env map[string]string) {
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

func (r *TaskRuntime) invokeJSRuntime(ctx context.Context, blob []byte, buffer io.Writer, env map[string]string) {
	modcache, ok := r.cache.Get(r.task.ID)
	if !ok {
		modcache = wazero.NewCompilationCache()
		slog.Warn("no cache hit", "task", r.task.ID)
		r.cache.Put(r.task.ID, modcache)
	}
	config := wazero.NewRuntimeConfig().WithCompilationCache(modcache)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	defer runtime.Close(ctx)

	mod, err := runtime.CompileModule(ctx, spidermonkey.WasmBlob)
	if err != nil {
		panic(err)
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)
	modConfig := wazero.NewModuleConfig().
		WithStdin(os.Stdin).
		WithStdout(buffer).
		WithArgs("", "-e", string(blob))
	_, err = runtime.InstantiateModule(ctx, mod, modConfig)
	if err != nil {
		panic(err)
	}
}
