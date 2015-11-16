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
	// Mux is the thing we use to send messages from probe -> app.
	Mux AppMultiplexer

	// Pipes is the singleton collection of Pipes in this probe.
	Pipes = &PipeRegistry{
		pipes: map[string]xfer.Pipe{},
	}
)

// PipeRegistry holds a collection of pipes.
type PipeRegistry struct {
	sync.Mutex
	pipes map[string]xfer.Pipe
}

// AppMultiplexer gets an PipeHandler for a given probe/
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

	id := fmt.Sprintf("pipe-%d", rand.Int63())
	pipe := xfer.NewPipe(id, handler)
	r.pipes[id] = pipe
	return pipe, nil
}
