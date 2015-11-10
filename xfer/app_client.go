package xfer

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/weaveworks/scope/common/sanitize"
)

// AppClient is a client for the scope app.
type AppClient interface {
	Details() (Details, error)
	ControlConnection()
	HandlePipeIO(*PipeIO) error
	Stop()
}

// Details are some generic details that can be fetched from /api
type Details struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// appClient is a client to an app for dealing with controls.
type appClient struct {
	ProbeConfig

	quit     chan struct{}
	target   string
	insecure bool
	client   http.Client

	controlServerCodecMtx sync.Mutex
	controlServerCodec    *JSONWebsocketCodec

	control ControlHandler
	pipe    PipeHandler
}

// NewAppClient makes a new appClient.
func NewAppClient(pc ProbeConfig, hostname, target string, control ControlHandler, pipe PipeHandler) (AppClient, error) {
	httpTransport, err := pc.getHTTPTransport(hostname)
	if err != nil {
		return nil, err
	}

	return &appClient{
		ProbeConfig: pc,
		quit:        make(chan struct{}),
		target:      target,
		client: http.Client{
			Transport: httpTransport,
		},
		control: control,
		pipe:    pipe,
	}, nil
}

// Stop stops the appClient.
func (c *appClient) Stop() {
	c.controlServerCodecMtx.Lock()
	defer c.controlServerCodecMtx.Unlock()
	close(c.quit)
	if c.controlServerCodec != nil {
		c.controlServerCodec.Close()
	}
	c.client.Transport.(*http.Transport).CloseIdleConnections()
}

// Details fetches the details (version, id) of the app.
func (c *appClient) Details() (Details, error) {
	result := Details{}
	req, err := c.ProbeConfig.authorizedRequest("GET", sanitize.URL("", 0, "/api")(c.target), nil)
	if err != nil {
		return result, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

func (c *appClient) HandlePipeIO(pio *PipeIO) error {
	c.controlServerCodecMtx.Lock()
	defer c.controlServerCodecMtx.Unlock()

	if c.controlServerCodec == nil {
		return fmt.Errorf("Not connected")
	}

	return c.controlServerCodec.HandlePipeIO(pio)
}

func (c *appClient) controlConnection() error {
	dialer := websocket.Dialer{}
	headers := http.Header{}
	c.ProbeConfig.authorizeHeaders(headers)
	// TODO(twilkie) need to update sanitize to work with wss
	url := sanitize.URL("ws://", 0, "/api/control/ws")(c.target)
	conn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return err
	}
	defer func() {
		log.Printf("Closing control connection to %s", c.target)
		conn.Close()
	}()

	codec := NewJSONWebsocketCodec(conn, c.pipe)
	server := rpc.NewServer()
	if err := server.RegisterName("control", c.control); err != nil {
		return err
	}

	c.controlServerCodecMtx.Lock()
	c.controlServerCodec = codec
	// At this point we may have tried to quit earlier, so check to see if the
	// quit channel has been closed, non-blocking.
	select {
	default:
	case <-c.quit:
		codec.Close()
		return nil
	}
	c.controlServerCodecMtx.Unlock()

	server.ServeCodec(codec)

	c.controlServerCodecMtx.Lock()
	c.controlServerCodec = nil
	c.controlServerCodecMtx.Unlock()
	return nil
}

func (c *appClient) controlConnectionLoop() {
	defer log.Printf("Control connection to %s exiting", c.target)
	backoff := initialBackoff

	for {
		err := c.controlConnection()
		if err == nil {
			backoff = initialBackoff
			continue
		}

		log.Printf("Error doing controls for %s, backing off %s: %v", c.target, backoff, err)
		select {
		case <-time.After(backoff):
		case <-c.quit:
			return
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (c *appClient) ControlConnection() {
	go c.controlConnectionLoop()
}
