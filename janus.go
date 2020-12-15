// Package janus is a Golang implementation of the Janus API, used to interact
// with the Janus WebRTC Gateway.
package janus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/segmentio/ksuid"
	"nhooyr.io/websocket"
)

var debug = false

func unexpected(request string) error {
	return fmt.Errorf("Unexpected response received to '%s' request", request)
}

func newRequest(method string) (map[string]interface{}, chan interface{}) {
	req := make(map[string]interface{}, 8)
	req["janus"] = method
	return req, make(chan interface{})
}

type Transaction struct {
	ID           string
	StartedAt    time.Time
	Used         bool
	ResponseChan chan interface{}
}

type transactions struct {
	transactions map[string]*Transaction
	sync.Mutex
}

func (t *transactions) Add(id string, transaction *Transaction) {
	t.Lock()
	t.transactions[id] = transaction
	t.Unlock()
}

func (t *transactions) Get(id string) (*Transaction, error) {
	t.Lock()
	defer t.Unlock()
	if transaction, found := t.transactions[id]; found == true {
		return transaction, nil
	}
	return nil, fmt.Errorf("Transaction '%s' not found", id)
}

func (t *transactions) Delete(id string) error {
	t.Lock()
	defer t.Unlock()
	if _, found := t.transactions[id]; found == true {
		delete(t.transactions, id)
		return nil
	}
	return fmt.Errorf("Transaction '%s' not found", id)
}

// Gateway represents a connection to an instance of the Janus Gateway.
type Gateway struct {
	// Sessions is a map of the currently active sessions to the gateway.
	Sessions map[uint64]*Session

	// Access to the Sessions map should be synchronized with the Gateway.Lock()
	// and Gateway.Unlock() methods provided by the embeded sync.Mutex.
	sync.Mutex

	conn         *websocket.Conn
	transactions *transactions
	errors       chan error
	shutdown     chan bool
	sendChan     chan []byte
	writeMu      sync.Mutex
}

// Connect initiates a webscoket connection with the Janus Gateway
func Connect(ctx context.Context, wsURL string) (*Gateway, error) {
	opts := &websocket.DialOptions{Subprotocols: []string{"janus-protocol"}}

	conn, _, err := websocket.Dial(ctx, wsURL, opts)

	if err != nil {
		return nil, err
	}

	gateway := new(Gateway)
	gateway.conn = conn
	gateway.transactions = &transactions{transactions: map[string]*Transaction{}}
	gateway.Sessions = make(map[uint64]*Session)
	gateway.sendChan = make(chan []byte, 100)
	gateway.errors = make(chan error)
	gateway.shutdown = make(chan bool)

	go gateway.ping(ctx)
	go gateway.recv(ctx)
	return gateway, nil
}

// Close closes the underlying connection to the Gateway.
func (gateway *Gateway) Close(code websocket.StatusCode, reason string) error {
	gateway.shutdown <- false
	return gateway.conn.Close(code, reason)
}

// GetErrChan returns a channels through which the caller can check and react to connectivity errors
func (gateway *Gateway) GetErrChan() chan error {
	return gateway.errors
}

func (gateway *Gateway) send(ctx context.Context, msg map[string]interface{}, responseChan chan interface{}) {
	id := ksuid.New()
	transaction := &Transaction{ID: id.String(), ResponseChan: responseChan, Used: false, StartedAt: time.Now()}

	msg["transaction"] = transaction.ID
	gateway.transactions.Add(transaction.ID, transaction)

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("json.Marshal: %s\n", err)
		return
	}

	if debug {
		// log message being sent
		var log bytes.Buffer
		json.Indent(&log, data, ">", "   ")
		log.Write([]byte("\n"))
		log.WriteTo(os.Stdout)
	}

	fmt.Println(".... sending transaction ", id)

	gateway.writeMu.Lock()
	err = gateway.conn.Write(ctx, websocket.MessageText, data)
	gateway.writeMu.Unlock()

	if err != nil {
		select {
		case gateway.errors <- err:
		default:
			fmt.Printf("conn.Write: %s\n", err)
		}

		return
	}
}

func passMsg(ch chan interface{}, msg interface{}) {
	ch <- msg
}

func (gateway *Gateway) ping(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-gateway.shutdown:
			return
		case <-ticker.C:
			err := gateway.conn.Write(ctx, 9, []byte{})
			if err != nil {
				select {
				case gateway.errors <- err:
				default:
					log.Println("ping:", err)
				}

				return
			}
		}
	}
}

func (gateway *Gateway) recv(ctx context.Context) {

	for {

		var base BaseMsg

		// Read message from Gateway
		_, data, err := gateway.conn.Read(ctx)
		if err != nil {
			select {
			case gateway.errors <- err:
			default:
				fmt.Printf("conn.Read: %s\n", err)
			}

			return
		}

		// Decode to Msg struct
		if err := json.Unmarshal(data, &base); err != nil {
			fmt.Printf("json.Unmarshal: %s\n", err)
			continue
		}

		if debug {
			// log message being sent
			var log bytes.Buffer
			json.Indent(&log, data, "<", "   ")
			log.Write([]byte("\n"))
			log.WriteTo(os.Stdout)
		}

		typeFunc, ok := msgtypes[base.Type]
		if !ok {
			fmt.Printf("Unknown message type received!\n")
			continue
		}

		msg := typeFunc()
		if err := json.Unmarshal(data, &msg); err != nil {
			fmt.Printf("json.Unmarshal: %s\n", err)
			continue // Decode error
		}

		var transaction *Transaction
		if base.ID != "" {
			transaction, _ = gateway.transactions.Get(base.ID)
		}

		// Pass message on from here
		if transaction == nil {
			// Is this a Handle event?
			if base.Handle == 0 {
				// Error()
			} else {
				// Lookup Session
				gateway.Lock()
				session := gateway.Sessions[base.Session]
				gateway.Unlock()
				if session == nil {
					fmt.Printf("Unable to deliver message. Session gone?\n")
					continue
				}

				// Lookup Handle
				session.Lock()
				handle := session.Handles[base.Handle]
				session.Unlock()
				if handle == nil {
					fmt.Printf("Unable to deliver message. Handle gone?\n")
					continue
				}

				// Pass msg
				go passMsg(handle.Events, msg)
			}
		} else {
			switch msg.(type) {
			case *EventMsg:
				gateway.transactions.Delete(transaction.ID)
			}

			fmt.Println(".... received transaction ", transaction.ID)

			// Pass msg
			go passMsg(transaction.ResponseChan, msg)
		}
	}
}

// Info sends an info request to the Gateway.
// On success, an InfoMsg will be returned and error will be nil.
func (gateway *Gateway) Info(ctx context.Context) (*InfoMsg, error) {
	req, ch := newRequest("info")
	gateway.send(ctx, req, ch)

	msg := <-ch
	switch msg := msg.(type) {
	case *InfoMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("info")
}

// Create sends a create request to the Gateway.
// On success, a new Session will be returned and error will be nil.
func (gateway *Gateway) Create(ctx context.Context) (*Session, error) {
	req, ch := newRequest("create")
	gateway.send(ctx, req, ch)

	msg := <-ch
	var success *SuccessMsg
	switch msg := msg.(type) {
	case *SuccessMsg:
		success = msg
	case *ErrorMsg:
		return nil, msg
	}

	// Create new session
	session := new(Session)
	session.gateway = gateway
	session.ID = success.Data.ID
	session.Handles = make(map[uint64]*Handle)
	session.Events = make(chan interface{}, 2)

	// Store this session
	gateway.Lock()
	gateway.Sessions[session.ID] = session
	gateway.Unlock()

	return session, nil
}

// Session represents a session instance on the Janus Gateway.
type Session struct {
	// ID is the session_id of this session
	ID uint64

	// Handles is a map of plugin handles within this session
	Handles map[uint64]*Handle

	Events chan interface{}

	// Access to the Handles map should be synchronized with the Session.Lock()
	// and Session.Unlock() methods provided by the embeded sync.Mutex.
	sync.Mutex

	gateway *Gateway
}

func (session *Session) send(ctx context.Context, msg map[string]interface{}, responseChan chan interface{}) {
	msg["session_id"] = session.ID
	session.gateway.send(ctx, msg, responseChan)
}

// Attach sends an attach request to the Gateway within this session.
// plugin should be the unique string of the plugin to attach to.
// On success, a new Handle will be returned and error will be nil.
func (session *Session) Attach(ctx context.Context, plugin string) (*Handle, error) {
	req, ch := newRequest("attach")
	req["plugin"] = plugin
	session.send(ctx, req, ch)

	var success *SuccessMsg
	msg := <-ch
	switch msg := msg.(type) {
	case *SuccessMsg:
		success = msg
	case *ErrorMsg:
		return nil, msg
	}

	handle := new(Handle)
	handle.session = session
	handle.ID = success.Data.ID
	handle.Events = make(chan interface{}, 8)

	session.Lock()
	session.Handles[handle.ID] = handle
	session.Unlock()

	return handle, nil
}

// KeepAlive sends a keep-alive request to the Gateway.
// On success, an AckMsg will be returned and error will be nil.
func (session *Session) KeepAlive(ctx context.Context) (*AckMsg, error) {
	req, ch := newRequest("keepalive")
	session.send(ctx, req, ch)

	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("keepalive")
}

// Destroy sends a destroy request to the Gateway to tear down this session.
// On success, the Session will be removed from the Gateway.Sessions map, an
// AckMsg will be returned and error will be nil.
func (session *Session) Destroy(ctx context.Context) (*AckMsg, error) {
	req, ch := newRequest("destroy")
	session.send(ctx, req, ch)

	var ack *AckMsg
	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		ack = msg
	case *ErrorMsg:
		return nil, msg
	}

	// Remove this session from the gateway
	session.gateway.Lock()
	delete(session.gateway.Sessions, session.ID)
	session.gateway.Unlock()

	return ack, nil
}

// Handle represents a handle to a plugin instance on the Gateway.
type Handle struct {
	// ID is the handle_id of this plugin handle
	ID uint64

	// Type   // pub  or sub
	Type string

	//User   // Userid
	User string

	// Events is a receive only channel that can be used to receive events
	// related to this handle from the gateway.
	Events chan interface{}

	session *Session
}

func (handle *Handle) send(ctx context.Context, msg map[string]interface{}, responseChan chan interface{}) {
	msg["handle_id"] = handle.ID
	handle.session.send(ctx, msg, responseChan)
}

// Request sends a sync request
func (handle *Handle) Request(ctx context.Context, body interface{}) (*SuccessMsg, error) {
	req, respCh := newRequest("message")
	if body != nil {
		req["body"] = body
	}
	handle.send(ctx, req, respCh)

	fmt.Println(".... waiting for message on ", handle.ID)

	msg := <-respCh

	fmt.Println(".... received message", handle.ID)

	switch msg := msg.(type) {
	case *SuccessMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("message")
}

// Message sends a message request to a plugin handle on the Gateway.
// body should be the plugin data to be passed to the plugin, and jsep should
// contain an optional SDP offer/answer to establish a WebRTC PeerConnection.
// On success, an EventMsg will be returned and error will be nil.
func (handle *Handle) Message(ctx context.Context, body, jsep interface{}) (*EventMsg, error) {
	req, ch := newRequest("message")
	if body != nil {
		req["body"] = body
	}
	if jsep != nil {
		req["jsep"] = jsep
	}
	handle.send(ctx, req, ch)

GetMessage: // No tears..
	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		goto GetMessage // ..only dreams.
	case *EventMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("message")
}

// Trickle sends a trickle request to the Gateway as part of establishing
// a new PeerConnection with a plugin.
// candidate should be a single ICE candidate, or a completed object to
// signify that all candidates have been sent:
//		{
//			"completed": true
//		}
// On success, an AckMsg will be returned and error will be nil.
func (handle *Handle) Trickle(ctx context.Context, candidate interface{}) (*AckMsg, error) {
	req, ch := newRequest("trickle")
	req["candidate"] = candidate
	handle.send(ctx, req, ch)

	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("trickle")
}

// TrickleMany sends a trickle request to the Gateway as part of establishing
// a new PeerConnection with a plugin.
// candidates should be an array of ICE candidates.
// On success, an AckMsg will be returned and error will be nil.
func (handle *Handle) TrickleMany(ctx context.Context, candidates interface{}) (*AckMsg, error) {
	req, ch := newRequest("trickle")
	req["candidates"] = candidates
	handle.send(ctx, req, ch)

	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("trickle")
}

// Detach sends a detach request to the Gateway to remove this handle.
// On success, an AckMsg will be returned and error will be nil.
func (handle *Handle) Detach(ctx context.Context) (*AckMsg, error) {
	req, ch := newRequest("detach")
	handle.send(ctx, req, ch)

	var ack *AckMsg
	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		ack = msg
	case *ErrorMsg:
		return nil, msg
	}

	// Remove this handle from the session
	handle.session.Lock()
	delete(handle.session.Handles, handle.ID)
	handle.session.Unlock()

	return ack, nil
}
