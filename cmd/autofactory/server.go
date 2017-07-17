package main

import (
	"net/url"
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/cmd/autofactory/output"
	"github.com/mohae/autofact/conf"
	"github.com/mohae/autofact/db"
	"github.com/mohae/autofact/message"
	"github.com/mohae/autofact/util"
	"github.com/mohae/joefriday/cpu/cpuutil/flat"
	"github.com/mohae/joefriday/net/netusage/flat"
	"github.com/mohae/joefriday/sysinfo/loadavg/flat"
	"github.com/mohae/joefriday/sysinfo/mem/flat"
	"github.com/mohae/randchars"
	"github.com/mohae/snoflinga"
	"github.com/mohae/systeminfo"
	czap "github.com/mohae/zap"
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
	influxUser    string
	influxPass    string
	idGen         snoflinga.Generator
	TSLayout      string //the layout for timestamps
	UseTS         bool   // TODO work out how this should be used; currentyl, it's a bit haphazard.
}

func newServer() *server {
	return &server{
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

func (s *server) SetInfluxDB(u, p string) error {
	s.influxUser = u
	s.influxPass = p
	// connect to Influx
	err := s.connectToInfluxDB()
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "connect to influxdb"),
			zap.String("db", s.InfluxDBName),
		)
		return err
	}
	// start the Influx writer
	// TODO: influx writer should handle done channel signaling
	go s.InfluxClient.Write()

	return nil
}

// connects to InfluxDB
func (s *server) connectToInfluxDB() error {
	var err error
	s.InfluxClient, err = newInfluxClient(s.InfluxDBName, s.InfluxAddress, s.influxUser, s.influxPass)
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
		tsLayout:     s.TSLayout,
		useTS:        s.UseTS,
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
		if !s.Inventory.clientExists(id) {
			c = s.newClient(id)
			s.Inventory.clients[string(id)] = c.Conf
			c.InfluxClient = s.InfluxClient
			break
		}
	}
	// save the client info to the db
	err = s.Bolt.SaveClient(c.Conf)
	c.tsLayout = s.TSLayout
	c.useTS = s.UseTS
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
	if data != nil {
		c.Data = data.With(
			czap.String("client", string(c.Conf.IDBytes())),
		)
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
	isConnected    bool
	CPUUtilization func(*message.Message)
	LoadAvg        func(*message.Message)
	MemInfo        func(*message.Message)
	NetUsage       func(*message.Message)
	tsLayout       string //the layout for timestamps
	useTS          bool
	// Data is a child Data Logger with relevant context for when output is to a File.
	Data czap.Logger
}

// SetFuncs sets the processing func for the client based on the output destination type.
func (c *Client) SetFuncs() {
	// at this point outputType is a supported output.Type so only need to handle
	// the supported ones.
	switch outputType {
	case output.File:
		c.CPUUtilization = c.CPUUtilizationFile
		c.LoadAvg = c.LoadAvgFile
		c.MemInfo = c.MemInfoFile
		c.NetUsage = c.NetUsageFile
	case output.InfluxDB:
		c.CPUUtilization = c.CPUUtilizationInfluxDB
		c.LoadAvg = c.LoadAvgInfluxDB
		c.MemInfo = c.MemInfoInfluxDB
		c.NetUsage = c.NetUsageFile
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
			if _, ok := err.(*websocket.CloseError); !ok {
				log.Error(
					err.Error(),
					zap.String("op", "read message"),
					zap.String("client", string(c.Conf.IDBytes())),
				)
				return
			}
			log.Info(
				err.Error(),
				zap.String("op", "read message"),
				zap.String("client", string(c.Conf.IDBytes())),
			)
			log.Warn(
				"client closed connection...waiting for reconnect",
				zap.String("op", "read message"),
				zap.String("client", string(c.Conf.IDBytes())),
			)
			return
		}
		switch typ {
		case websocket.TextMessage:
			// Currently, no text message are expected so warn.
			// TODO is just logging it as a warn the correct course of action here?
			log.Warn(
				string(p),
				zap.String("op", "receive message"),
				zap.String("client", string(c.Conf.IDBytes())),
				zap.String("type", util.WSString(typ)),
			)

		case websocket.BinaryMessage:
			c.processBinaryMessage(p)
		case websocket.CloseMessage:
			log.Info(
				string(p),
				zap.String("op", "client closed connection"),
				zap.String("client", string(c.Conf.IDBytes())),
				zap.String("type", util.WSString(typ)),
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
				log.Error(
					err.Error(),
					zap.String("op", "health request"),
					zap.String("client", string(c.Conf.IDBytes())),
				)
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
		log.Debug(
			"cpuutil",
			zap.String("client", string(c.Conf.Hostname())),
		)
		c.CPUUtilization(msg)
	case message.LoadAvg:
		log.Debug(
			"loadavg",
			zap.String("client", string(c.Conf.Hostname())),
		)
		c.LoadAvg(msg)
	case message.MemInfo:
		log.Debug(
			"meminfo",
			zap.String("client", string(c.Conf.Hostname())),
		)
		c.MemInfo(msg)
	case message.NetUsage:
		log.Debug(
			"netusage",
			zap.String("client", string(c.Conf.Hostname())),
		)
		c.NetUsage(msg)
	case message.SysInfoJSON:
		log.Debug(
			"sysinfojson",
			zap.String("client", string(c.Conf.Hostname())),
		)
		s, err := systeminfo.JSONUnmarshal(p)
		if err != nil {
			log.Error(
				err.Error(),
				zap.String("op", "unmarshal JSON"),
				zap.String("client", string(c.Conf.IDBytes())),
				zap.String("kind", k.String()),
			)
			return nil
		}
		// TODO: should something be done with sysinfo, other than log it?
		log.Info(
			"systeminfo",
			zap.String("client", string(c.Conf.Hostname())),
			zap.Object("data", s),
		)
	default:
		log.Error(
			"unsupported message kind",
			zap.String("op", "process binary message"),
			zap.String("client", string(c.Conf.IDBytes())),
			zap.String("kind", k.String()),
			zap.Base64("message", p),
		)
	}
	return nil
}

// CPUUtilizationInfluxDB processes CPUUtilization messages and saves to
// InfluxDB
func (c *Client) CPUUtilizationInfluxDB(msg *message.Message) {
	cpus := cpuutil.Deserialize(msg.DataBytes())
	tags := map[string]string{"host": string(c.Conf.Hostname()), "region": string(c.Conf.Region())}
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
			log.Error(
				err.Error(),
				zap.String("op", "create point"),
				zap.String("client", string(c.Conf.IDBytes())),
				zap.String("stat", "cpu utilization"),
				zap.String("cpu", cpu.ID),
			)
			continue
		}
		pts = append(pts, pt)
	}
	// only send if there were any points generated
	if len(pts) != 0 {
		c.InfluxClient.pointsCh <- pts
	}
}

// CPUUtilizationInfluxDB processes CPUUtilization messages and saves to file
// as JSON.
func (c *Client) CPUUtilizationFile(msg *message.Message) {
	// using Object means that timestamp will be a replicated field, as ts and
	// timestamp, with timestamp being the int64 because I don't want to write
	// a separate entry per cpu and replicate the top level delta info.
	cpus := cpuutil.Deserialize(msg.DataBytes())
	if c.useTS {
		c.Data.Info(
			"cpuutil",
			// add timestamp handling
			czap.Int64("ts", cpus.Timestamp),
			czap.Object("data", cpus),
		)
		return
	}
	c.Data.Info(
		"cpuutil",
		// add timestamp handling
		czap.String("ts", c.FormattedTime(cpus.Timestamp)),
		czap.Object("data", cpus),
	)

}

// LoadAvgInfluxDB processes LoadAvg messages and saves to InfluxDB.
func (c *Client) LoadAvgInfluxDB(msg *message.Message) {
	l := loadavg.Deserialize(msg.DataBytes())
	tags := map[string]string{"host": string(c.Conf.Hostname()), "region": string(c.Conf.Region())}
	fields := map[string]interface{}{
		"one":     l.One,
		"five":    l.Five,
		"fifteen": l.Fifteen,
	}
	pt, err := influx.NewPoint("loadavg", tags, fields, time.Unix(0, l.Timestamp).UTC())
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "create point"),
			zap.String("client", string(c.Conf.IDBytes())),
			zap.String("stat", "loadavg"),
		)
	} else {
		c.InfluxClient.pointsCh <- []*influx.Point{pt}
	}
}

// LoadAvgFile processes LoadAvg messages and saves to the data file as JSON.
func (c *Client) LoadAvgFile(msg *message.Message) {
	l := loadavg.Deserialize(msg.DataBytes())
	c.Data.Info(
		"loadavg",
		czap.String("ts", c.FormattedTime(l.Timestamp)),
		czap.Float64("one", l.One),
		czap.Float64("five", l.Five),
		czap.Float64("fifteen", l.Fifteen),
	)
}

// MemInfoInfluxDB processes MemInfo messages and saves to InfluxDB.
func (c *Client) MemInfoInfluxDB(msg *message.Message) {
	m := mem.Deserialize(msg.DataBytes())
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
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "create point"),
			zap.String("client", string(c.Conf.IDBytes())),
			zap.String("stat", "meminfo"),
		)
	} else {
		c.InfluxClient.pointsCh <- []*influx.Point{pt}
	}
}

// MemInfoFile processes MemInfo messages and saves to file as JSON.
func (c *Client) MemInfoFile(msg *message.Message) {
	m := mem.Deserialize(msg.DataBytes())
	c.Data.Info(
		"meminfo",
		czap.String("ts", c.FormattedTime(m.Timestamp)),
		czap.Uint64("total_ram", m.TotalRAM),
		czap.Uint64("free_ram", m.FreeRAM),
		czap.Uint64("shared_ram", m.SharedRAM),
		czap.Uint64("buffer_ram", m.BufferRAM),
		czap.Uint64("total_swap", m.TotalSwap),
		czap.Uint64("free_swap", m.FreeSwap),
	)
}

// NetUsageInfluxDB processes NetUSage messages and saves them to InfluxDB
func (c *Client) NetUsageInfluxDB(msg *message.Message) {
	devs := netusage.Deserialize(msg.DataBytes())
	tags := map[string]string{"host": string(c.Conf.Hostname()), "region": string(c.Conf.Region())}
	// Make a slice of points whose length is equal to the number of Interfaces
	// and process the interfaces.
	pts := make([]*influx.Point, 0, len(devs.Device))
	for _, dev := range devs.Device {
		tags["device"] = string(dev.Name)
		fields := map[string]interface{}{
			"received.bytes":         dev.RBytes,
			"received.packets":       dev.RPackets,
			"received.errs":          dev.RErrs,
			"received.drop":          dev.RDrop,
			"received.fifo":          dev.RFIFO,
			"received.frame":         dev.RFrame,
			"received.compressed":    dev.RCompressed,
			"received.multicast":     dev.RMulticast,
			"transmitted.bytes":      dev.TBytes,
			"transmitted.packets":    dev.TPackets,
			"transmitted.errs":       dev.TErrs,
			"transmitted.drop":       dev.TDrop,
			"transmitted.fifo":       dev.TFIFO,
			"transmitted.colls":      dev.TColls,
			"transmitted.carrier":    dev.TCarrier,
			"transmitted.compressed": dev.TCompressed,
		}
		pt, err := influx.NewPoint("interfaces", tags, fields, time.Unix(0, devs.Timestamp).UTC())
		if err != nil {
			log.Error(
				err.Error(),
				zap.String("op", "create point"),
				zap.String("client", string(c.Conf.IDBytes())),
				zap.String("stat", "netusage"),
				zap.String("device", string(dev.Name)),
			)
			continue
		}
		pts = append(pts, pt)
	}
	// only send if there were any points generated
	if len(pts) > 0 {
		c.InfluxClient.pointsCh <- pts
	}
}

// NetUsageFile processes NetUsage messages and writes it to the data file as
// JSON. Each interface is it's own entry
func (c *Client) NetUsageFile(msg *message.Message) {
	devs := netusage.Deserialize(msg.DataBytes())
	for _, dev := range devs.Device {
		ts := c.FormattedTime(devs.Timestamp)
		c.Data.Info(
			"netusage",
			czap.String("ts", ts),
			czap.Int64("tdelta", devs.TimeDelta),
			czap.String("name", dev.Name),
			czap.Int64("rbytes", dev.RBytes),
			czap.Int64("rpackets", dev.RPackets),
			czap.Int64("rerrs", dev.RErrs),
			czap.Int64("rdrop", dev.RDrop),
			czap.Int64("rfifo", dev.RFIFO),
			czap.Int64("rframe", dev.RFrame),
			czap.Int64("rcompressed", dev.RCompressed),
			czap.Int64("tmulticast", dev.RMulticast),
			czap.Int64("tbytes", dev.TBytes),
			czap.Int64("tpackets", dev.TPackets),
			czap.Int64("terrs", dev.TErrs),
			czap.Int64("tdrop", dev.TDrop),
			czap.Int64("tfifo", dev.TFIFO),
			czap.Int64("tcolls", dev.TColls),
			czap.Int64("tcarrier", dev.TCarrier),
			czap.Int64("rcompressed", dev.TCompressed),
		)
	}
}

// FormattedTime returns the nanoseconds as a formatted datetime string using
// the client's layout.
func (c *Client) FormattedTime(t int64) string {
	return time.Unix(0, t).Format(c.tsLayout)
}
