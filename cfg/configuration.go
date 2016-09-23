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
	ServerAddress   string        `json:"server_address"`
	ServerPort      string        `json:"server_port"`
	ServerID        uint32        `json:"server_id"`
	ConnectInterval time.Duration `json:"connect_interval"`
	ConnectPeriod   time.Duration `json:"connect_period"`
	filename        string
	Conf
}

// LoadConn loads the config file.  The Conn's filename is set during this
// operation.
// TODO: revisit this design decision.
func (c *Conn) Load(name string) error {
	c.filename = name
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return fmt.Errorf("error unmarshaling confg file %s: %s", name, err)
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

// LoadNodeInf loads the cfg.NodeInf from the file specified.  If it doesn't
// exist, a NodeInf with its ID set to 0 and the current hostname is returned.
func LoadNodeInf(name string) (*NodeInf, error) {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		bldr := flatbuffers.NewBuilder(0)
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("load node inf failed: could not determine hostname: %s\n", err)
		}
		h := bldr.CreateString(hostname)
		NodeInfStart(bldr)
		NodeInfAddID(bldr, 0)
		NodeInfAddHostname(bldr, h)
		bldr.Finish(NodeInfEnd(bldr))
		return GetRootAsNodeInf(bldr.Bytes[bldr.Head():], 0), nil
	}
	return GetRootAsNodeInf(b, 0), nil
}

// Serialize serializes the Node using flatbuffers and returns the []byte.
func (n *NodeInf) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	h := bldr.CreateByteString(n.Hostname())
	r := bldr.CreateByteString(n.Region())
	z := bldr.CreateByteString(n.Zone())
	d := bldr.CreateByteString(n.DataCenter())
	NodeInfStart(bldr)
	NodeInfAddID(bldr, n.ID())
	NodeInfAddHostname(bldr, h)
	NodeInfAddRegion(bldr, r)
	NodeInfAddZone(bldr, z)
	NodeInfAddDataCenter(bldr, d)
	bldr.Finish(NodeInfEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Save the current Node to a file.
func (n *NodeInf) Save(fname string) error {
	return ioutil.WriteFile(fname, n.Serialize(), 0600)
}

// Serialize serializes the struct.  The flatbuffers definition for this
// struct is in autofact/cfg_clientConf.fbs and the resulting definition is in
// cfg/ClientConf.go
func (c *ClientConf) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	ClientConfStart(bldr)
	ClientConfAddHealthbeatInterval(bldr, c.HealthbeatInterval())
	ClientConfAddHealthbeatPushPeriod(bldr, c.HealthbeatPushPeriod())
	ClientConfAddSaveInterval(bldr, c.SaveInterval())
	bldr.Finish(ClientConfEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Deserialize deserializes the bytes into the current Conf.
func (c *ClientConf) Deserialize(p []byte) {
	c = GetRootAsClientConf(p, 0)
}
