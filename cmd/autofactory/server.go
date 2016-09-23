package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/cfg"
	"github.com/mohae/autofact/db"
	"github.com/mohae/autofact/message"
	cpuutil "github.com/mohae/joefriday/cpu/utilization/flat"
	netf "github.com/mohae/joefriday/net/usage/flat"
	loadf "github.com/mohae/joefriday/sysinfo/load/flat"
	memf "github.com/mohae/joefriday/sysinfo/mem/flat"
)

// Defaults for ClientCfg: if file doesn't exist.
var (
	DefaultHealthbeatInterval   = 5 * time.Second
	DefaultHealthbeatPushPeriod = 15 * time.Second
	DefaultSaveInterval         = 30 * time.Second
)

// server is the container for a server's information and everything that it
// is tracking/serving.
type server struct {
	// Autofact directory path
	AutoPath string
	// ID of the server
	ID uint32
	// URL of the server
	url.URL
	// Flatbuffers serialized default client config.
	ClientConf []byte
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
	nodes, err := s.DB.Nodes()
	if err != nil {
		return n, err
	}
	for i, c := range nodes {
		s.Inventory.AddNode(c.ID(), c)
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

// Client checks the inventory to see if the client exists.  If it exists,
// a Client is created from the client in the inventory.
func (s *server) Client(id uint32) (*Client, bool) {
	c, ok := s.Inventory.Node(id)
	if !ok {
		return nil, false
	}
	return &Client{
		NodeInf:      c,
		ClientConf:   cfg.GetRootAsClientConf(clientConf, 0),
		InfluxClient: s.InfluxClient,
	}, true
}

// NewClient creates a new Node, adds it to the server's inventory and
// returns the client.Inf to the caller.   If the save of the Client's inf to
// the database results in an error, it will be returned.
func (s *server) NewClient() (*Client, error) {
	// get a new client
	c := s.Inventory.NewNode()
	c.InfluxClient = s.InfluxClient
	// save the client info to the db
	err := s.DB.SaveNode(c.NodeInf)
	return c, err
}

// Client holds information about a client.
type Client struct {
	*cfg.NodeInf
	*cfg.ClientConf
	WS *websocket.Conn
	*InfluxClient
	isConnected bool
}

func newClient(id uint32) *Client {
	bldr := flatbuffers.NewBuilder(0)
	cfg.NodeInfStart(bldr)
	cfg.NodeInfAddID(bldr, id)
	bldr.Finish(cfg.NodeInfEnd(bldr))
	return &Client{
		NodeInf:    cfg.GetRootAsNodeInf(bldr.Bytes[bldr.Head():], 0),
		ClientConf: cfg.GetRootAsClientConf(clientConf, 0),
	}
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
	c.WS.WriteMessage(websocket.BinaryMessage, message.Serialize(c.NodeInf.ID(), k, p))
}

// binary messages are expected to be flatbuffer encoding of message.Message.
// TODO:  revisit design of tag and field handling; make pluggable for
// backends other than influx?  Make more flexible, perhaps funcs to call or
// define interface(s)
func (c *Client) processBinaryMessage(p []byte) error {
	// unmarshal the message
	msg := message.GetRootAsMessage(p, 0)
	// process according to kind
	k := message.Kind(msg.Kind())
	switch k {
	case message.CPUUtilization:
		fmt.Printf("%s: cpu utilization\n", c.NodeInf.Hostname())
		cpus := cpuutil.Deserialize(msg.DataBytes())
		tags := map[string]string{"host": string(c.NodeInf.Hostname()), "region": string(c.NodeInf.Region())}
		var bErr error // this is the last error in the batch, if any
		// Each cpu is it's own point, make a slice to accommodate them all and process.
		pts := make([]*influx.Point, 0, len(cpus.CPU))
		for _, cpu := range cpus.CPU {
			tags["cpu"] = cpu.ID
			fields := map[string]interface{}{
				"usage":  float32(cpu.Usage) / 100.0,
				"user":   float32(cpu.User) / 100.0,
				"nice":   float32(cpu.Nice) / 100.0,
				"system": float32(cpu.System) / 100.0,
				"idle":   float32(cpu.Idle) / 100.0,
				"iowait": float32(cpu.IOWait) / 100.0,
			}
			pt, err := influx.NewPoint("cpus", tags, fields, time.Unix(0, cpus.Timestamp).UTC())
			if err != nil {
				fmt.Fprintf(os.Stderr, "cpu utilization: create infulx.Point: %s: %s", cpu.ID, err)
				bErr = err
			}
			pts = append(pts, pt)
		}
		c.InfluxClient.seriesCh <- Series{Data: pts, err: bErr}
	case message.SysLoadAvg:
		fmt.Printf("%s: loadavg\n", c.NodeInf.Hostname())
		l := loadf.Deserialize(msg.DataBytes())
		tags := map[string]string{"host": string(c.NodeInf.Hostname()), "region": string(c.NodeInf.Region())}
		fields := map[string]interface{}{
			"one":     l.One,
			"five":    l.Five,
			"fifteen": l.Fifteen,
		}
		pt, err := influx.NewPoint("loadavg", tags, fields, time.Unix(0, l.Timestamp).UTC())
		c.InfluxClient.seriesCh <- Series{Data: []*influx.Point{pt}, err: err}
	case message.SysMemInfo:
		fmt.Printf("%s: meminfo\n", c.NodeInf.Hostname())
		m := memf.Deserialize(msg.DataBytes())
		tags := map[string]string{"host": string(c.NodeInf.Hostname()), "region": string(c.NodeInf.Region())}
		fields := map[string]interface{}{
			"total_ram":  m.TotalRAM,
			"free_ram":   m.FreeRAM,
			"shared_ram": m.SharedRAM,
			"buffer_ram": m.BufferRAM,
			"total_swap": m.TotalSwap,
			"free_swap":  m.FreeSwap,
		}
		pt, err := influx.NewPoint("memory", tags, fields, time.Unix(0, m.Timestamp).UTC())
		c.InfluxClient.seriesCh <- Series{Data: []*influx.Point{pt}, err: err}
	case message.NetUsage:
		fmt.Printf("%s: network usage\n", c.NodeInf.Hostname())
		ifaces := netf.Deserialize(msg.DataBytes())
		tags := map[string]string{"host": string(c.NodeInf.Hostname()), "region": string(c.NodeInf.Region())}
		var bErr error // the last error in the batch, if any
		// Make a slice of points whose length is equal to the number of Interfaces
		// and process the interfaces.
		pts := make([]*influx.Point, 0, len(ifaces.Interfaces))
		for _, iFace := range ifaces.Interfaces {
			tags["interface"] = string(iFace.Name)
			fields := map[string]interface{}{
				"received.bytes":         iFace.RBytes,
				"received.packets":       iFace.RPackets,
				"received.errs":          iFace.RErrs,
				"received.drop":          iFace.RDrop,
				"received.fifo":          iFace.RFIFO,
				"received.frame":         iFace.RFrame,
				"received.compressed":    iFace.RCompressed,
				"received.multicast":     iFace.RMulticast,
				"transmitted.bytes":      iFace.TBytes,
				"transmitted.packets":    iFace.TPackets,
				"transmitted.errs":       iFace.TErrs,
				"transmitted.drop":       iFace.TDrop,
				"transmitted.fifo":       iFace.TFIFO,
				"transmitted.colls":      iFace.TColls,
				"transmitted.carrier":    iFace.TCarrier,
				"transmitted.compressed": iFace.TCompressed,
			}
			pt, err := influx.NewPoint("interfaces", tags, fields, time.Unix(0, ifaces.Timestamp).UTC())
			if err != nil {
				fmt.Fprintf(os.Stderr, "network interface usage: create influx.Point: %s: %s", iFace.Name, err)
				bErr = err
			}
			pts = append(pts, pt)
		}
		c.InfluxClient.seriesCh <- Series{Data: pts, err: bErr}
	default:
		fmt.Println("unknown message kind")
		fmt.Println(string(p))
	}
	return nil
}

// ClientCfg defines the client behavior, outside of connections.  This
// is serverside only and for loading the default clientCfg from a file.
// All communication of cfg data between Server and Client (Node) is done
// with Flatbuffers serialized cfg.Client.
type ClientCfg struct {
	RawHealthbeatInterval   string        `json:"healthbeat_interval"`
	HealthbeatInterval      time.Duration `json:"-"`
	RawHealthbeatPushPeriod string        `json:"healthbeat_push_period"`
	HealthbeatPushPeriod    time.Duration `json:"-"`
	RawSaveInterval         string        `json:"save_interval"`
	SaveInterval            time.Duration `json:"-"`
	WriteWait               time.Duration `json:"-"`
}

// Load loads the client configuration from the specified file.
func (c *ClientCfg) Load(cfgFile string) error {
	b, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, c)
	if err != nil {
		return fmt.Errorf("error unmarshaling client cfg file %s: %s", cfgFile, err)
	}
	c.HealthbeatInterval, err = time.ParseDuration(c.RawHealthbeatInterval)
	if err != nil {
		return fmt.Errorf("error parsing healthbeat interval: %s", err)
	}
	c.HealthbeatPushPeriod, err = time.ParseDuration(c.RawHealthbeatPushPeriod)
	if err != nil {
		return fmt.Errorf("error parsing healthbeat push period %s", err)
	}
	c.SaveInterval, err = time.ParseDuration(c.RawSaveInterval)
	if err != nil {
		return fmt.Errorf("error parsing save interval %s", err)
	}
	return nil
}

// Returns a ClientCfg with application defaults.  This is called when
// the Cfg file cannot be found.
func (c *ClientCfg) UseAppDefaults() {
	c.RawHealthbeatInterval = DefaultHealthbeatInterval.String()
	c.HealthbeatInterval = DefaultHealthbeatInterval
	c.RawHealthbeatPushPeriod = DefaultHealthbeatPushPeriod.String()
	c.HealthbeatPushPeriod = DefaultHealthbeatPushPeriod
	c.RawSaveInterval = DefaultSaveInterval.String()
	c.SaveInterval = DefaultSaveInterval
	// WriteWait isn't set because it isn't being used yet.
}

func (c *ClientCfg) SaveAsJSON(fname string) error {
	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return fmt.Errorf("error marshaling ClientCfg to JSON: %s", err)
	}
	err = ioutil.WriteFile(fname, b, 0600)
	if err != nil {
		return fmt.Errorf("ClientCfg save error: %s", err)
	}
	return nil
}

// Serialize serializes the struct as a cfg.Conf.
func (c *ClientCfg) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	cfg.ClientConfStart(bldr)
	cfg.ClientConfAddHealthbeatInterval(bldr, int64(c.HealthbeatInterval))
	cfg.ClientConfAddHealthbeatPushPeriod(bldr, int64(c.HealthbeatPushPeriod))
	cfg.ClientConfAddSaveInterval(bldr, int64(c.SaveInterval))
	bldr.Finish(cfg.ClientConfEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Deserialize deserializes serialized cfg.Conf into ClientCfg.
func (c *ClientCfg) Deserialize(p []byte) {
	conf := cfg.GetRootAsClientConf(p, 0)
	c.HealthbeatInterval = time.Duration(conf.HealthbeatInterval())
	c.HealthbeatPushPeriod = time.Duration(conf.HealthbeatPushPeriod())
	c.SaveInterval = time.Duration(conf.SaveInterval())
}
