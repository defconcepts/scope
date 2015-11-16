package xfer

import (
	"fmt"
	"io"
	"net/rpc"
	"sync"

	"github.com/gorilla/websocket"
)

// Request is the UI -> App -> Probe message type for control RPCs
type Request struct {
	AppID   string
	NodeID  string
	Control string
}

// Response is the Probe -> App -> UI message type for the control RPCs.
type Response struct {
	Value interface{} `json:"value,omitempty"`
	Error string      `json:"error,omitempty"`
	Pipe  string      `json:"pipe,omitempty"`
}

// PipeIO is athe Probe <-> App message type for pipes.
type PipeIO struct {
	ID  string
	Buf []byte
}

// Message is the unions of Request, Response and PipeIO
type Message struct {
	Request  *rpc.Request
	Response *rpc.Response
	Value    interface{}
	Pipe     *PipeIO
}

// PipeHandler is something that handles PipeIO messages
type PipeHandler interface {
	HandlePipeIO(*PipeIO) error
}

// PipeHandlerFunc is an adapter (ala golang's http RequestHandlerFunc) for
// PipeHandler.
type PipeHandlerFunc func(*PipeIO) error

// HandlePipeIO implements PipeHandler
func (f PipeHandlerFunc) HandlePipeIO(pio *PipeIO) error {
	return f(pio)
}

// ControlHandler is interface used in the app and the probe to represent
// a control RPC.
type ControlHandler interface {
	Handle(req Request, res *Response) error
}

// ControlHandlerFunc is a adapter (ala golang's http RequestHandlerFunc)
// for ControlHandler
type ControlHandlerFunc func(Request) Response

// Handle is an adapter method to make ControlHandlers exposable via golang rpc
func (c ControlHandlerFunc) Handle(req Request, res *Response) error {
	*res = c(req)
	return nil
}

// ResponseErrorf creates a new Response with the given formatted error string.
func ResponseErrorf(format string, a ...interface{}) Response {
	return Response{
		Error: fmt.Sprintf(format, a...),
	}
}

// ResponseError creates a new Response with the given error.
func ResponseError(err error) Response {
	if err != nil {
		return Response{
			Error: err.Error(),
		}
	}
	return Response{}
}

// JSONWebsocketCodec is golang rpc compatible Server and Client Codec
// that transmits and receives RPC messages over a websocker, as JSON.
type JSONWebsocketCodec struct {
	sync.Mutex
	conn *websocket.Conn
	err  chan struct{}
	pipe PipeHandler
}

// NewJSONWebsocketCodec makes a new JSONWebsocketCodec
func NewJSONWebsocketCodec(conn *websocket.Conn, pipe PipeHandler) *JSONWebsocketCodec {
	return &JSONWebsocketCodec{
		conn: conn,
		err:  make(chan struct{}),
		pipe: pipe,
	}
}

// WaitForReadError blocks until any read on this codec returns an error.
// This is useful to know when the server has disconnected from the client.
func (j *JSONWebsocketCodec) WaitForReadError() {
	<-j.err
}

// WriteRequest implements rpc.ClientCodec
func (j *JSONWebsocketCodec) WriteRequest(r *rpc.Request, v interface{}) error {
	j.Lock()
	defer j.Unlock()

	if err := j.conn.WriteJSON(Message{Request: r}); err != nil {
		return err
	}
	return j.conn.WriteJSON(Message{Value: v})
}

// WriteResponse implements rpc.ServerCodec
func (j *JSONWebsocketCodec) WriteResponse(r *rpc.Response, v interface{}) error {
	j.Lock()
	defer j.Unlock()

	if err := j.conn.WriteJSON(Message{Response: r}); err != nil {
		return err
	}
	return j.conn.WriteJSON(Message{Value: v})
}

// HandlePipeIO writes a pipe io to the websocket
func (j *JSONWebsocketCodec) HandlePipeIO(p *PipeIO) error {
	j.Lock()
	defer j.Unlock()
	return j.conn.WriteJSON(Message{Pipe: p})
}

func (j *JSONWebsocketCodec) readMessage(v interface{}) (*Message, error) {
	for {
		m := Message{Value: v}
		if err := j.conn.ReadJSON(&m); err != nil {
			close(j.err)
			return nil, err
		}

		if m.Pipe != nil {
			j.pipe.HandlePipeIO(m.Pipe)
			continue
		}

		return &m, nil
	}
}

// ReadResponseHeader implements rpc.ClientCodec
func (j *JSONWebsocketCodec) ReadResponseHeader(r *rpc.Response) error {
	m, err := j.readMessage(nil)
	if err == nil {
		*r = *m.Response
	}
	return err
}

// ReadResponseBody implements rpc.ClientCodec
func (j *JSONWebsocketCodec) ReadResponseBody(v interface{}) error {
	_, err := j.readMessage(v)
	return err
}

// Close implements rpc.ClientCodec and rpc.ServerCodec
func (j *JSONWebsocketCodec) Close() error {
	return j.conn.Close()
}

// ReadRequestHeader implements rpc.ServerCodec
func (j *JSONWebsocketCodec) ReadRequestHeader(r *rpc.Request) error {
	m, err := j.readMessage(nil)
	if err == nil {
		*r = *m.Request
	}
	return err
}

// ReadRequestBody implements rpc.ServerCodec
func (j *JSONWebsocketCodec) ReadRequestBody(v interface{}) error {
	_, err := j.readMessage(v)
	return err
}

// Pipe is a bi-directional channel from someone thing in the probe
// to the UI.
type Pipe interface {
	ID() string

	PipeHandler
	io.Reader
	io.Writer
	io.Closer
}

type pipe struct {
	id string

	pw             *io.PipeWriter // for the HandlePipeIO implementation
	*io.PipeReader                // for the Read() implementation
	handler        PipeHandler    // used by Write()
}

func NewPipe(id string, write PipeHandler) Pipe {
	r, w := io.Pipe()
	return &pipe{
		id:         id,
		handler:    write,
		PipeReader: r,
		pw:         w,
	}
}

func (p *pipe) ID() string {
	return p.id
}

func (p *pipe) HandlePipeIO(pio *PipeIO) error {
	// this is an incoming pio from the app, so is a write to this pipe
	// that needs buffering until a corresponding call to Read()
	_, err := p.pw.Write(pio.Buf)
	return err
}

func (p *pipe) Close() error {
	return nil // TODO close pipe, etc
}

func (p *pipe) Write(b []byte) (n int, err error) {
	if err := p.handler.HandlePipeIO(&PipeIO{
		ID:  p.id,
		Buf: b,
	}); err != nil {
		return 0, err
	}

	return len(b), nil
}
