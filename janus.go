// Package janus is a Golang implementation of the Janus API, used to interact
// with the Janus WebRTC Gateway.
package janus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var debug = false

func init() {
	if os.Getenv("DEBUG") != "" {
		debug = true
	}
}

func unexpected(request string) error {
	return fmt.Errorf("Unexpected response received to '%s' request", request)
}

func newRequest(method string) (map[string]interface{}, chan interface{}) {
	req := make(map[string]interface{}, 8)
	req["janus"] = method
	return req, make(chan interface{})
}

// Gateway represents a connection to an instance of the Janus Gateway.
type Gateway struct {
	// Sessions is a map of the currently active sessions to the gateway.
	Sessions map[uint64]*Session

	// Access to the Sessions map should be synchronized with the Gateway.Lock()
	// and Gateway.Unlock() methods provided by the embeded sync.Mutex.
	sync.Mutex

	conn            *websocket.Conn
	nextTransaction uint64
	transactions    map[uint64]chan interface{}
}

// Connect creates a new Gateway instance, connected to the Janus Gateway.
// path should be a filesystem path to the Unix Socket that the Unix transport
// is bound to.
// On success, a new Gateway object will be returned and error will be nil.
// func Connect(path string, netType string) (*Gateway, error) {
// 	conn, err := net.Dial(netType, path)
// 	if err != nil {
// 		return nil, err
// 	}

// 	gateway := new(Gateway)
// 	//gateway.conn = conn
// 	gateway.transactions = make(map[uint64]chan interface{})
// 	gateway.Sessions = make(map[uint64]*Session)

// 	go gateway.recv()
// 	return gateway, nil
// }

func Connect(wsURL string) (*Gateway, error) {
	websocket.DefaultDialer.Subprotocols = []string{"janus-protocol"}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)

	if err != nil {
		return nil, err
	}

	gateway := new(Gateway)
	gateway.conn = conn
	gateway.transactions = make(map[uint64]chan interface{})
	gateway.Sessions = make(map[uint64]*Session)

	go gateway.ping()
	go gateway.recv()
	return gateway, nil
}

// Close closes the underlying connection to the Gateway.
func (gateway *Gateway) Close() error {
	return gateway.conn.Close()
}

func (gateway *Gateway) send(msg map[string]interface{}, transaction chan interface{}) {
	id := atomic.AddUint64(&gateway.nextTransaction, 1)

	msg["transaction"] = strconv.FormatUint(id, 10)
	gateway.Lock()
	gateway.transactions[id] = transaction
	gateway.Unlock()

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

	err = gateway.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		fmt.Printf("conn.Write: %s\n", err)
		return
	}
}

func passMsg(ch chan interface{}, msg interface{}) {
	ch <- msg
}

func (gateway *Gateway) ping() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := gateway.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Println("ping:", err)
				return
			}
		}
	}
}

func (gateway *Gateway) recv() {

	for {
		// Read message from Gateway

		// Decode to Msg struct
		var base BaseMsg

		_, data, err := gateway.conn.ReadMessage()
		if err != nil {
			fmt.Printf("conn.Read: %s\n", err)
			return
		}

		if err := json.Unmarshal(data, &base); err != nil {
			fmt.Printf("json.Unmarshal: %s\n", err)
			continue
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

		// Pass message on from here
		if base.Id == "" {
			// Is this a Handle event?
			if base.Handle == 0 {
				// Nope. No idea what's going on...
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
			id, _ := strconv.ParseUint(base.Id, 10, 64) // FIXME: error checking
			// Lookup Transaction
			gateway.Lock()
			transaction := gateway.transactions[id]
			gateway.Unlock()
			if transaction == nil {
				// Error()
			}

			// Pass msg
			go passMsg(transaction, msg)
		}
	}
}

// Info sends an info request to the Gateway.
// On success, an InfoMsg will be returned and error will be nil.
func (gateway *Gateway) Info() (*InfoMsg, error) {
	req, ch := newRequest("info")
	gateway.send(req, ch)

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
func (gateway *Gateway) Create() (*Session, error) {
	req, ch := newRequest("create")
	gateway.send(req, ch)

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
	session.Id = success.Data.Id
	session.Handles = make(map[uint64]*Handle)

	// Store this session
	gateway.Lock()
	gateway.Sessions[session.Id] = session
	gateway.Unlock()

	return session, nil
}

// Session represents a session instance on the Janus Gateway.
type Session struct {
	// Id is the session_id of this session
	Id uint64

	// Handles is a map of plugin handles within this session
	Handles map[uint64]*Handle

	// Access to the Handles map should be synchronized with the Session.Lock()
	// and Session.Unlock() methods provided by the embeded sync.Mutex.
	sync.Mutex

	gateway *Gateway
}

func (session *Session) send(msg map[string]interface{}, transaction chan interface{}) {
	msg["session_id"] = session.Id
	session.gateway.send(msg, transaction)
}

// Attach sends an attach request to the Gateway within this session.
// plugin should be the unique string of the plugin to attach to.
// On success, a new Handle will be returned and error will be nil.
func (session *Session) Attach(plugin string) (*Handle, error) {
	req, ch := newRequest("attach")
	req["plugin"] = plugin
	session.send(req, ch)

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
	handle.Id = success.Data.Id
	handle.Events = make(chan interface{}, 8)

	session.Lock()
	session.Handles[handle.Id] = handle
	session.Unlock()

	return handle, nil
}

// KeepAlive sends a keep-alive request to the Gateway.
// On success, an AckMsg will be returned and error will be nil.
func (session *Session) KeepAlive() (*AckMsg, error) {
	req, ch := newRequest("keepalive")
	session.send(req, ch)

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
func (session *Session) Destroy() (*AckMsg, error) {
	req, ch := newRequest("destroy")
	session.send(req, ch)

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
	delete(session.gateway.Sessions, session.Id)
	session.gateway.Unlock()

	return ack, nil
}

// Handle represents a handle to a plugin instance on the Gateway.
type Handle struct {
	// Id is the handle_id of this plugin handle
	Id uint64

	// Events is a receive only channel that can be used to receive events
	// related to this handle from the gateway.
	Events chan interface{}

	session *Session
}

func (handle *Handle) send(msg map[string]interface{}, transaction chan interface{}) {
	msg["handle_id"] = handle.Id
	handle.session.send(msg, transaction)
}

// Message sends a message request to a plugin handle on the Gateway.
// body should be the plugin data to be passed to the plugin, and jsep should
// contain an optional SDP offer/answer to establish a WebRTC PeerConnection.
// On success, an EventMsg will be returned and error will be nil.
func (handle *Handle) Message(body, jsep interface{}) (*EventMsg, error) {
	req, ch := newRequest("message")
	if body != nil {
		req["body"] = body
	}
	if jsep != nil {
		req["jsep"] = jsep
	}
	handle.send(req, ch)

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
func (handle *Handle) Trickle(candidate interface{}) (*AckMsg, error) {
	req, ch := newRequest("trickle")
	req["candidate"] = candidate
	handle.send(req, ch)

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
func (handle *Handle) TrickleMany(candidates interface{}) (*AckMsg, error) {
	req, ch := newRequest("trickle")
	req["candidates"] = candidates
	handle.send(req, ch)

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
func (handle *Handle) Detach() (*AckMsg, error) {
	req, ch := newRequest("detach")
	handle.send(req, ch)

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
	delete(handle.session.Handles, handle.Id)
	handle.session.Unlock()

	return ack, nil
}
