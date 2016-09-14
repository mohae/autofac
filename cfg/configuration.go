package cfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/google/flatbuffers/go"
)

// Conn holds the connection information for a node.  This is all that is
// persisted on a client node.
type Conn struct {
	ServerAddress      string        `json:"server_address"`
	ServerPort         string        `json:"server_port"`
	ServerID           uint32        `json:"server_id"`
	RawConnectInterval string        `json:"connect_interval"`
	ConnectInterval    time.Duration `json:"-"`
	RawConnectPeriod   string        `json:"connect_period"`
	ConnectPeriod      time.Duration `json:"-"`
	filename           string
}

// LoadConn loads the config file.  Errors are logged but not returned.
// TODO: revisit this design decision.
func (c *Conn) Load(name string) error {
	c.filename = name
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read of config file failed: %s", err)
	}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return fmt.Errorf("error unmarshaling confg file %s: %s", name, err)
	}
	c.ConnectInterval, err = time.ParseDuration(c.RawConnectInterval)
	if err != nil {
		return fmt.Errorf("error parsing connect interval: %s", err)
	}
	c.ConnectPeriod, err = time.ParseDuration(c.RawConnectPeriod)
	if err != nil {
		return fmt.Errorf("error parsing connect period: %s", err)
	}
	return nil
}

func (c *Conn) Save() error {
	j, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return fmt.Errorf("fail: marshal conn cfg to JSON: %s\n", err)
	}
	f, err := os.OpenFile(c.filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640)
	if err != nil {
		return fmt.Errorf("fail: conn cfg save: %s\n", err)
	}
	defer f.Close()
	n, err := f.Write(j)
	if err != nil {
		return fmt.Errorf("fail: conn cfg save: %s\n", err)
	}
	if n != len(j) {
		return fmt.Errorf("fail: conn cfg save: short write: wrote %d of %d bytes\n", n, len(j))
	}
	return nil
}

func (c *Conn) SetFilename(v string) {
	c.filename = v
}

// LoadNode loads the cfg.Node from the received file.  If it doesn't exist
// a basic inf with its ID set to 0  and current hostname set is returned.
func LoadNode(name string) (*Node, error) {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		bldr := flatbuffers.NewBuilder(0)
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("load node inf failed: could not determine hostname: %s\n", err)
		}
		h := bldr.CreateString(hostname)
		NodeStart(bldr)
		NodeAddID(bldr, 0)
		NodeAddHostname(bldr, h)
		bldr.Finish(NodeEnd(bldr))
		return GetRootAsNode(bldr.Bytes[bldr.Head():], 0), nil
	}
	return GetRootAsNode(b, 0), nil
}

// Serialize serializes the Node using flatbuffers and returns the []byte.
func (n *Node) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	h := bldr.CreateByteString(n.Hostname())
	r := bldr.CreateByteString(n.Region())
	z := bldr.CreateByteString(n.Zone())
	d := bldr.CreateByteString(n.DataCenter())
	NodeStart(bldr)
	NodeAddID(bldr, n.ID())
	NodeAddHostname(bldr, h)
	NodeAddRegion(bldr, r)
	NodeAddZone(bldr, z)
	NodeAddDataCenter(bldr, d)
	bldr.Finish(NodeEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Save the current Node to a file.
func (n *Node) Save(fname string) error {
	return ioutil.WriteFile(fname, n.Serialize(), 0600)
}

// Serialize serializes the struct.  The flatbuffers definition for this
// struct is in autofact/cfg_conf.fbs and the resulting definition is in
// cfg/Conf.go
func (c *Conf) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	ConfStart(bldr)
	ConfAddHealthbeatInterval(bldr, c.HealthbeatInterval())
	ConfAddHealthbeatPushPeriod(bldr, c.HealthbeatPushPeriod())
	ConfAddPingPeriod(bldr, c.PingPeriod())
	ConfAddPongWait(bldr, c.PongWait())
	ConfAddSaveInterval(bldr, c.SaveInterval())
	bldr.Finish(ConfEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Deserialize deserializes the bytes into the current Conf.
func (c *Conf) Deserialize(p []byte) {
	c = GetRootAsConf(p, 0)
}
