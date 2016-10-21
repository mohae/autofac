package main

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/conf"
	"github.com/mohae/autofact/db"
	"github.com/mohae/autofact/message"
	"github.com/mohae/autofact/util"
	cpuutil "github.com/mohae/joefriday/cpu/utilization/flat"
	netf "github.com/mohae/joefriday/net/usage/flat"
	loadf "github.com/mohae/joefriday/sysinfo/load/flat"
	memf "github.com/mohae/joefriday/sysinfo/mem/flat"
	"github.com/mohae/randchars"
	"github.com/mohae/snoflinga"
	"github.com/uber-go/zap"
)

// server is the container for a server's information and everything that it
// is tracking/serving.
type server struct {
	// Autofact directory path
	AutoPath string `json:"-"`
	// ID of the server
	ID []byte `json:"id"`
	// URL of the server
	url.URL `json:"-"`
	// Default client config.
	conf.Collect `json:"-"`
	// A map of clients, by ID
	Inventory inventory `json:"-"`
	// TODO: add handling to prevent the same client from connecting
	// more than once:  this requires detection of reconnect of an
	// existing client vs an existing client maintaining multiple
	// con-current connections
	db.Bolt `json:"-"`
	// InfluxDB client
	*InfluxClient `json:"-"`
	// DB info.
	// TODO: should this be persisted; if not, remove the json tags
	BoltDBFile    string `json:"bolt_db_file"`
	InfluxDBName  string `json:"influx_db_name"`
	InfluxAddress string `json:"influx_address"`
	idGen         snoflinga.Generator
}

func newServer() server {
	return server{
		Inventory: newInventory(),
	}
}

// NewSnowflakeGenerator gets a new snowflake generator for message id
// generation.
func (s *server) NewSnowflakeGenerator() {
	s.idGen = snoflinga.New(s.ID)
}

// LoadInventory populates the server's inventory from the database.  This
// is a cached list of clients.
// TODO: should the client configs be cached or should they be read from
// a config file?  Any client that has a healthbeat or metric collection
// configuration different than the standard would probably need to be in a
// config file, unless some other mechanism was provided (get from etcd or
// something?)
func (s *server) LoadInventory() (int, error) {
	var n int
	clients, err := s.Bolt.Clients()
	if err != nil {
		return n, err
	}
	for i, c := range clients {
		s.Inventory.AddClient(c)
		n = i
	}
	return n, nil
}

// connects to InfluxDB
func (s *server) connectToInfluxDB(u, p string) error {
	var err error
	s.InfluxClient, err = newInfluxClient(srvr.InfluxDBName, srvr.InfluxAddress, u, p)
	return err
}

// Client checks the inventory to see if the client exists.  If it exists,
// a Client is created from the client in the inventory.
func (s *server) Client(id []byte) (*Client, bool) {
	c, ok := s.Inventory.Client(id)
	if !ok {
		return nil, false
	}
	return &Client{
		Conf:         c,
		InfluxClient: s.InfluxClient,
	}, true
}

// NewClient creates a new Node, adds it to the server's inventory and
// returns the client.Inf to the caller.   If the save of the Client's inf to
// the database results in an error, it will be returned.
func (s *server) NewClient() (c *Client, err error) {
	// get a new client
	s.Inventory.mu.Lock()
	defer s.Inventory.mu.Unlock()
	for {
		// TODO replace with a rand bytes or striing
		id := randchars.AlphaNum(util.IDLen)
		fmt.Println(string(id))
		if !s.Inventory.clientExists(id) {
			c = s.newClient(id)
			s.Inventory.clients[string(id)] = c.Conf
			c.InfluxClient = s.InfluxClient
			break
		}
	}
	// save the client info to the db
	err = s.Bolt.SaveClient(c.Conf)
	return c, err
}

func (s *server) newClient(id []byte) *Client {
	bldr := flatbuffers.NewBuilder(0)
	v := bldr.CreateByteVector(id)
	conf.ClientStart(bldr)
	conf.ClientAddID(bldr, v)
	conf.ClientAddHealthbeatPeriod(bldr, s.HealthbeatPeriod.Int64())
	conf.ClientAddMemInfoPeriod(bldr, s.MemInfoPeriod.Int64())
	conf.ClientAddCPUUtilizationPeriod(bldr, s.CPUUtilizationPeriod.Int64())
	conf.ClientAddNetUsagePeriod(bldr, s.NetUsagePeriod.Int64())
	bldr.Finish(conf.ClientEnd(bldr))
	c := Client{
		Conf: conf.GetRootAsClient(bldr.Bytes[bldr.Head():], 0),
	}

	return &c
}

// WriteBinaryMessage serializes a message and writes it to the socket as
// a binary message.
func (s *server) WriteBinaryMessage(client string, conn *websocket.Conn, k message.Kind, p []byte) {
	err := conn.WriteMessage(websocket.BinaryMessage, message.Serialize(s.idGen.Snowflake(), k, p))
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "write binary message"),
			zap.String("client", client),
			zap.String("kind", k.String()),
			zap.Base64("message", p),
		)
	}
}

// Client holds information about a client.
type Client struct {
	Conf *conf.Client
	WS   *websocket.Conn
	*InfluxClient
	isConnected bool
}

// Listen listens for messages and handles them accordingly.  Binary messages
// are expected to be  Flatbuffer serialized bytes containing a Message.
func (c *Client) Listen(doneCh chan struct{}) {
	// loop until there's a done signal
	defer close(doneCh)
	for {
		typ, p, err := c.WS.ReadMessage()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); !ok {
				log.Error(
					err.Error(),
					zap.String("op", "read message"),
					zap.String("id", string(c.Conf.IDBytes())),
				)
				return
			}
			log.Info(
				err.Error(),
				zap.String("op", "read message"),
				zap.String("id", string(c.Conf.IDBytes())),
			)

			fmt.Println("client closed connection...waiting for reconnect")
			return
		}
		switch typ {
		case websocket.TextMessage:
			// Currently, no text message are expected so warn.
			// TODO is just logging it as a warn the correct course of action here?
			log.Warn(
				string(p),
				zap.String("op", "receive message"),
				zap.String("type", util.WSString(typ)),
			)

		case websocket.BinaryMessage:
			c.processBinaryMessage(p)
		case websocket.CloseMessage:
			log.Info(
				string(p),
				zap.String("op", "client closed connection"),
				zap.String("type", util.WSString(typ)),
				zap.String("id", string(c.Conf.IDBytes())),
			)
			return
		}
	}
}

// Healthbeat pulls info from the client on a set interval.  If the client
// doesn't respond before the deadline, an error is generated.  If the client
// doesn't respond to several consecutive requests, an error is generated and
// the client connection is closed.
func (c *Client) Healthbeat(done chan struct{}) {
	// If this was set to 0; don't do a healthbeat.
	if c.Conf.HealthbeatPeriod() == 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(c.Conf.HealthbeatPeriod()))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// request the Healthbeat; serveClient will handle the response.
			err := c.WS.WriteMessage(websocket.TextMessage, autofact.LoadAvg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: healthbeat error: %s", string(c.Conf.IDBytes()), err)
				return
			} // add the data
		case <-done:
			return
		}
	}
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
		fmt.Printf("%s: cpu utilization\n", c.Conf.Hostname())
		cpus := cpuutil.Deserialize(msg.DataBytes())
		tags := map[string]string{"host": string(c.Conf.Hostname()), "region": string(c.Conf.Region())}
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
	case message.LoadAvg:
		fmt.Printf("%s: loadavg\n", c.Conf.Hostname())
		l := loadf.Deserialize(msg.DataBytes())
		tags := map[string]string{"host": string(c.Conf.Hostname()), "region": string(c.Conf.Region())}
		fields := map[string]interface{}{
			"one":     l.One,
			"five":    l.Five,
			"fifteen": l.Fifteen,
		}
		pt, err := influx.NewPoint("loadavg", tags, fields, time.Unix(0, l.Timestamp).UTC())
		c.InfluxClient.seriesCh <- Series{Data: []*influx.Point{pt}, err: err}
	case message.MemInfo:
		fmt.Printf("%s: meminfo\n", c.Conf.Hostname())
		m := memf.Deserialize(msg.DataBytes())
		tags := map[string]string{"host": string(c.Conf.Hostname()), "region": string(c.Conf.Region())}
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
		fmt.Printf("%s: network usage\n", c.Conf.Hostname())
		ifaces := netf.Deserialize(msg.DataBytes())
		tags := map[string]string{"host": string(c.Conf.Hostname()), "region": string(c.Conf.Region())}
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
