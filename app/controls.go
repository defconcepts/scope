package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/rpc"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/weaveworks/scope/xfer"
)

type controlHandler struct {
	id     int64
	client *rpc.Client
	codec  *xfer.JSONWebsocketCodec
}

type controlRouter struct {
	sync.Mutex
	probes map[string]controlHandler
	pipes  map[string]xfer.Pipe
}

func registerControlRoutes(router *mux.Router) {
	controlRouter := &controlRouter{
		probes: map[string]controlHandler{},
		pipes:  map[string]xfer.Pipe{},
	}
	router.Methods("GET").Path("/api/control/ws").HandlerFunc(controlRouter.handleProbeWS)
	router.Methods("GET").Path("/api/pipe/{pipeID}").HandlerFunc(controlRouter.handlePipeWS)
	router.Methods("POST").MatcherFunc(URLMatcher("/api/control/{probeID}/{nodeID}/{control}")).HandlerFunc(controlRouter.handleControl)
}

func (ch *controlHandler) handle(req xfer.Request) xfer.Response {
	var res xfer.Response
	if err := ch.client.Call("control.Handle", req, &res); err != nil {
		return xfer.ResponseError(err)
	}
	return res
}

func (ch *controlHandler) HandlePipeIO(pio *xfer.PipeIO) error {
	return ch.codec.HandlePipeIO(pio)
}

func (cr *controlRouter) get(probeID string) (controlHandler, bool) {
	cr.Lock()
	defer cr.Unlock()
	handler, ok := cr.probes[probeID]
	return handler, ok
}

func (cr *controlRouter) set(probeID string, handler controlHandler) {
	cr.Lock()
	defer cr.Unlock()
	cr.probes[probeID] = handler
}

func (cr *controlRouter) rm(probeID string, handler controlHandler) {
	cr.Lock()
	defer cr.Unlock()
	// NB probe might have reconnected in the mean time, need to ensure we do not
	// delete new connection!  Also, it might have connected then deleted itself!
	if cr.probes[probeID].id == handler.id {
		delete(cr.probes, probeID)
	}
}

// handleControl routes control requests from the client to the appropriate
// probe.  Its is blocking.
func (cr *controlRouter) handleControl(w http.ResponseWriter, r *http.Request) {
	var (
		vars    = mux.Vars(r)
		probeID = vars["probeID"]
		nodeID  = vars["nodeID"]
		control = vars["control"]
	)
	handler, ok := cr.get(probeID)
	if !ok {
		log.Printf("Probe %s is not connected right now...", probeID)
		http.NotFound(w, r)
		return
	}

	result := handler.handle(xfer.Request{
		AppID:   uniqueID,
		NodeID:  nodeID,
		Control: control,
	})
	if result.Error != "" {
		respondWith(w, http.StatusBadRequest, result.Error)
		return
	}
	if result.Pipe != "" {
		cr.getOrCreatePipe(result.Pipe, probeID)
	}
	respondWith(w, http.StatusOK, result)
}

// handleProbeWS accepts websocket connections from the probe and registers
// them in the control router, such that HandleControl calls can find them.
func (cr *controlRouter) handleProbeWS(w http.ResponseWriter, r *http.Request) {
	probeID := r.Header.Get(xfer.ScopeProbeIDHeader)
	if probeID == "" {
		respondWith(w, http.StatusBadRequest, xfer.ScopeProbeIDHeader)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to websocket: %v", err)
		return
	}
	defer conn.Close()

	pipeHandler := xfer.PipeHandlerFunc(func(pio *xfer.PipeIO) error {
		pipe := cr.getOrCreatePipe(pio.ID, probeID)
		return pipe.HandlePipeIO(pio)
	})
	codec := xfer.NewJSONWebsocketCodec(conn, pipeHandler)
	client := rpc.NewClientWithCodec(codec)
	handler := controlHandler{
		id:     rand.Int63(),
		codec:  codec,
		client: client,
	}

	cr.set(probeID, handler)

	codec.WaitForReadError()

	cr.rm(probeID, handler)
	client.Close()
}

func (cr *controlRouter) getOrCreatePipe(id string, probeID string) xfer.Pipe {
	cr.Lock()
	defer cr.Unlock()
	pipe, ok := cr.pipes[id]
	if !ok {
		log.Printf("Creating pipe id %s", id)
		handler := xfer.PipeHandlerFunc(func(pio *xfer.PipeIO) error {
			ch, ok := cr.get(probeID)
			if !ok {
				return fmt.Errorf("probe not found: %s", probeID)
			}
			return ch.HandlePipeIO(pio)
		})
		pipe = xfer.NewPipe(id, handler)
		cr.pipes[id] = pipe
	}
	return pipe
}

func (cr *controlRouter) getPipe(id string) (xfer.Pipe, bool) {
	cr.Lock()
	defer cr.Unlock()
	pipe, ok := cr.pipes[id]
	return pipe, ok
}

func (cr *controlRouter) handlePipeWS(w http.ResponseWriter, r *http.Request) {
	var (
		vars   = mux.Vars(r)
		pipeID = vars["pipeID"]
	)
	pipe, ok := cr.getPipe(pipeID)
	if !ok {
		log.Printf("Pipe %s is not connected right now...", pipeID)
		http.NotFound(w, r)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to websocket: %v", err)
		return
	}
	defer conn.Close()
	readQuit := make(chan struct{})
	writeQuit := make(chan struct{})

	// Read-from-UI loop
	go func() {
		defer close(readQuit)
		for {
			_, buf, err := conn.ReadMessage() // TODO type should be binary message
			if err != nil {
				log.Printf("Error reading websocket for pipe %s: %v", pipeID, err)
				return
			}

			if _, err := pipe.Write(buf); err != nil {
				log.Printf("Error writing pipe %s: %v", pipeID, err)
				return
			}
		}
	}()

	// Write-to-UI loop
	go func() {
		defer close(writeQuit)
		buf := make([]byte, 1024)
		for {
			n, err := pipe.Read(buf)
			if err != nil {
				log.Printf("Error reading pipe %s: %v", pipeID, err)
				return
			}

			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				log.Printf("Error writing websocket for pipe %s: %v", pipeID, err)
				return
			}
		}
	}()

	// block until one of the goroutines exits
	// this convoluted mechanism is to ensure we only close the websocket once.
	select {
	case <-readQuit:
	case <-writeQuit:
	}
}
