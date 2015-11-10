package controls

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/weaveworks/scope/xfer"
)

const (
	maxOutstandingPIOs = 32
)

var (
	Mux   AppMultiplexer
	Pipes = &PipeRegistry{
		pipes: map[int64]xfer.Pipe{},
	}
)

type PipeRegistry struct {
	sync.Mutex
	pipes map[int64]xfer.Pipe
}

type AppMultiplexer interface {
	PipeHandlerFor(string) xfer.PipeHandler
}

// HandlePipeIO handles incoming PipeIOs
func (r *PipeRegistry) HandlePipeIO(pio *xfer.PipeIO) error {
	r.Lock()
	defer r.Unlock()

	pipe, ok := r.pipes[pio.ID]
	if !ok {
		fmt.Errorf("Pipe ID %d not found", pio.ID)
	}

	return pipe.HandlePipeIO(pio)
}

// NewPipe destroys pipes.
func (r *PipeRegistry) NewPipe(appID string) (xfer.Pipe, error) {
	r.Lock()
	defer r.Unlock()

	handler := Mux.PipeHandlerFor(appID)
	if handler == nil {
		return nil, fmt.Errorf("No client connection for %s", appID)
	}

	pipe := xfer.NewPipe(rand.Int63(), handler)
	r.pipes[pipe.ID()] = pipe
	return pipe, nil
}
