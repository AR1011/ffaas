package actrs

import (
	"github.com/anthdm/ffaas/pkg/storage"
	"github.com/anthdm/ffaas/pkg/types"
	"github.com/anthdm/hollywood/actor"
)

const KindProcessRuntime = "process_runtime"

// ProcessRuntime is an actor that can execute compiled WASM blobs in a distributed cluster.
type ProcessRuntime struct {
	store   storage.Store
	cache   storage.ModCacher
	process *types.Process
}

func NewProcessRuntime(store storage.Store, cache storage.ModCacher) actor.Producer {
	return func() actor.Receiver {
		return &ProcessRuntime{
			store:   store,
			cache:   cache,
			process: nil,
		}
	}
}

func (r *ProcessRuntime) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
	case actor.Stopped:
	default:
		_ = msg

	}
}

// TODO
