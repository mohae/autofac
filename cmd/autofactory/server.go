package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/client"
	"github.com/mohae/autofact/db"
	"github.com/mohae/autofact/message"
	"github.com/mohae/autofact/sysinfo"
)

// server is the container for a server's information and everything that it
// is tracking/serving.
type server struct {
	// ID of the server
	ID uint32
	// URL of the server
	url.URL
	// Period between pings
	PingPeriod time.Duration
	// How long to wait for a pong response before timing out
	PongWait time.Duration
	// Flatbuffers serialized default client config
	ClientCfg []byte
	// A map of clients, by ID
	Inventory inventory
	// TODO: add handling to prevent the same client from connecting
	// more than once:  this requires detection of reconnect of an
	// existing client vs an existing client maintaining multiple
	// con-current connections
	DB db.Bolt
	// InfluxDB client
	*InfluxClient
}

func newServer(id uint32) server {
	return server{
		ID:        id,
		Inventory: newInventory(),
	}
}

// LoadInventory populates the server's inventory from the database.  This
// is a cached list of clients.
func (s *server) LoadInventory() (int, error) {
	var n int
	ids, err := s.DB.ClientIDs()
	if err != nil {
		return n, err
	}
	for i, id := range ids {
		c := newClient(id)
		c.InfluxClient = s.InfluxClient
		s.Inventory.AddClient(id, c)
		n = i
	}
	return n, nil
}

// connects to InfluxDB
func (s *server) connectToInfluxDB() error {
	var err error
	s.InfluxClient, err = newInfluxClient(influxDBName, influxAddress, influxUser, influxPassword)
	return err
}

// Client checks the inventory to see if the client exists
func (s *server) Client(id uint32) (*Client, bool) {
	return s.Inventory.Client(id)
}

// NewClient creates a new Client, adds it to the server's inventory and
// returns the client to the caller.   If the save of the Client's info to
// the database results in an error, it will be returned.
func (s *server) NewClient() (*Client, error) {
	// get a new client
	cl := s.Inventory.NewClient()
	cl.InfluxClient = s.InfluxClient
	// save the client info to the db
	err := s.DB.SaveClient(cl.ID)
	return cl, err
}

// Client holds the client's configuration, the websocket connection to the
// client node, and its connection state.
type Client struct {
	ID     uint32
	Name   string
	Region string
	client.Cfg
	WS *websocket.Conn
	*InfluxClient
	isConnected bool
}

// TODO: add region support (hardcoded for now for dev purposes)
func newClient(id uint32) *Client {
	return &Client{
		ID:     id,
		Name:   strconv.FormatUint(uint64(id), 10),
		Region: "region1",
		Cfg: client.Cfg{
			HealthbeatInterval:   clientCfg.HealthbeatInterval,
			HealthbeatPushPeriod: clientCfg.HealthbeatPushPeriod,
			PingPeriod:           clientCfg.PingPeriod,
			PongWait:             clientCfg.PongWait,
			SaveInterval:         clientCfg.SaveInterval,
			WriteWait:            clientCfg.WriteWait,
		},
	}
}

// PingHandler is the handler for Pings.
func (c *Client) PingHandler(msg string) error {
	fmt.Printf("ping: %s\n", msg)
	return c.WS.WriteMessage(websocket.PongMessage, []byte("ping"))
}

// PongHandler is the handler for pongs.
func (c *Client) PongHandler(msg string) error {
	fmt.Printf("pong: %s\n", msg)
	return c.WS.WriteMessage(websocket.PingMessage, []byte("pong"))
}

// Listen listens for messages and handles them accordingly.  Binary messages
// are expected to be  Flatbuffer serialized bytes containing a Message.
func (c *Client) Listen(doneCh chan struct{}) {
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

// WriteBinaryMessage serializes a message and writes it to the socket as
// a binary message.
func (c *Client) WriteBinaryMessage(k message.Kind, p []byte) {
	c.WS.WriteMessage(websocket.BinaryMessage, message.Serialize(c.ID, k, p))
}

// binary messages are expected to be flatbuffer encoding of message.Message.
func (c *Client) processBinaryMessage(p []byte) error {
	// unmarshal the message
	msg := message.GetRootAsMessage(p, 0)
	// process according to kind
	k := message.Kind(msg.Kind())
	switch k {
	case message.CPUData:
		cpu := sysinfo.GetRootAsCPUData(msg.DataBytes(), 0)
		tags := map[string]string{"host": c.Name, "region": c.Region, "cpu": string(cpu.CPUID())}
		fields := map[string]interface{}{
			"user":   float32(cpu.Usr()) / 100.0,
			"sys":    float32(cpu.Sys()) / 100.0,
			"iowait": float32(cpu.IOWait()) / 100.0,
			"idle":   float32(cpu.Idle()) / 100.0,
		}
		pt, err := influx.NewPoint("cpu_usage", tags, fields, time.Unix(0, cpu.Timestamp()).UTC())
		c.InfluxClient.seriesCh <- Series{Data: []*influx.Point{pt}, err: err}
		return nil
	case message.MemData:
		mem := sysinfo.GetRootAsMemData(msg.DataBytes(), 0)
		tags := map[string]string{"client": c.Name, "region": c.Region}
		fields := map[string]interface{}{
			"mem-total":   float32(mem.MemTotal()) / 100.0,
			"mem-used":    float32(mem.MemUsed()) / 100.0,
			"mem-free":    float32(mem.MemFree()) / 100.0,
			"mem-shared":  float32(mem.MemShared()) / 100.0,
			"mem-buffers": float32(mem.MemBuffers()) / 100.0,
			"cache-used":  float32(mem.CacheUsed()) / 100.0,
			"cache-free":  float32(mem.CacheFree()) / 100.0,
			"swap-total":  float32(mem.SwapTotal()) / 100.0,
			"swap-used":   float32(mem.SwapUsed()) / 100.0,
			"swap-free":   float32(mem.SwapFree()) / 100.0,
		}
		pt, err := influx.NewPoint("memory", tags, fields, time.Unix(0, mem.Timestamp()).UTC())
		c.InfluxClient.seriesCh <- Series{Data: []*influx.Point{pt}, err: err}
		return nil
	default:
		fmt.Println("unknown message kind")
		fmt.Println(string(p))
	}
	return nil
}
