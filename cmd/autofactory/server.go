package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/db"
	"github.com/mohae/autofact/message"
	"github.com/mohae/autofact/sysinfo"
)

type server struct {
	// ID of the server
	ID uint32
	// Interval between pings
	PingInterval time.Duration
	// How long to wait for a pong response before timing out
	PongWait time.Duration
	// A map of clients, by ID
	Inventory inventory
	// TODO: add handling to prevent the same client from connecting
	// more than once:  this requires detection of reconnect of an
	// existing client vs an existing client maintaining multiple
	// con-current connections
	DB db.Bolt
}

func newServer(id uint32) server {
	return server{
		ID:           id,
		PingInterval: autofact.PingPeriod,
		PongWait:     autofact.PongWait,
		Inventory:    newInventory(),
	}
}

// LoadInventory populates the inventory from the database.  This is a cached
// list of clients we are aware of.
func (s *server) LoadInventory() (int, error) {
	var n int
	ids, err := s.DB.ClientIDs()
	if err != nil {
		return n, err
	}
	for i, id := range ids {
		c := newClient(id)
		s.Inventory.AddClient(id, c)
		n = i
	}
	return n, nil
}

// Client checks the inventory to see if the client exists
func (s *server) Client(id uint32) (*client, bool) {
	return s.Inventory.Client(id)
}

// NewClient creates a new client.
func (s *server) NewClient() (*client, error) {
	// get a new client
	cl := s.Inventory.NewClient()
	// save the client info to the db
	err := s.DB.SaveClient(cl.Cfg.ID)
	return cl, err
}

// client holds the server side client info
type client struct {
	Cfg         autofact.ClientCfg
	WS          *websocket.Conn
	isConnected bool
}

func newClient(id uint32) *client {
	return &client{
		Cfg: autofact.ClientCfg{
			ID:         id,
			PingPeriod: autofact.PingPeriod,
			PongWait:   autofact.PongWait,
			WriteWait:  autofact.WriteWait,
		},
	}
}

func (c *client) PingHandler(msg string) error {
	fmt.Printf("ping: %s\n", msg)
	return c.WS.WriteMessage(websocket.PongMessage, []byte("ping"))
}

func (c *client) PongHandler(msg string) error {
	fmt.Printf("pong: %s\n", msg)
	return c.WS.WriteMessage(websocket.PingMessage, []byte("pong"))
}

func (c *client) Listen(doneCh chan struct{}) {
	// loop until there's a done signal
	defer close(doneCh)
	for {
		typ, p, err := c.WS.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading message: %s\n", err)
			if _, ok := err.(*websocket.CloseError); !ok {
				return
			}
			fmt.Println("client closed connection...waiting for reconnect")
			return
		}
		switch typ {
		case websocket.TextMessage:
			fmt.Printf("textmessage: %s\n", p)
			if bytes.Equal(p, autofact.AckMsg) {
				// if this is an acknowledgement message, do nothing
				continue
			}
			err := c.WS.WriteMessage(websocket.TextMessage, autofact.AckMsg)
			if err != nil {
				if _, ok := err.(*websocket.CloseError); !ok {
					return
				}
				fmt.Println("client closed connection...waiting for reconnect")
				return
			}
		case websocket.BinaryMessage:
			err = c.WS.WriteMessage(websocket.TextMessage, autofact.AckMsg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing binary message: %s\n", err)
				if _, ok := err.(*websocket.CloseError); !ok {
					return
				}
				fmt.Println("client closed connection...waiting for reconnect")
				return
			}
			c.processBinaryMessage(p)
		case websocket.CloseMessage:
			fmt.Printf("closemessage: %x\n", p)
			fmt.Println("client closed connection...waiting for reconnect")
			return
		}
	}
}

// binary messages are expected to be flatbuffer encoding of message.Message.
func (c *client) processBinaryMessage(p []byte) error {
	// unmarshal the message
	msg := message.GetRootAsMessage(p, 0)
	// process according to kind
	k := message.Kind(msg.Kind())
	switch k {
	case message.CPUData:
		fmt.Println(sysinfo.UnmarshalCPUDatasToString(msg.DataBytes()))
	case message.MemData:
		fmt.Println(sysinfo.UnmarshalMemDataToString(msg.DataBytes()))
	default:
		fmt.Println("unknown message kind")
		fmt.Println(string(p))
	}
	return nil
}
