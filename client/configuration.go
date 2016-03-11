package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/google/flatbuffers/go"
)

// ConnCfg holds the connection information for a node.  This is all that is
// persisted on a client node.
type ConnCfg struct {
	ServerAddress      string        `json:"server_address"`
	ServerPort         string        `json:"server_port"`
	ServerID           uint32        `json:"server_id"`
	RawConnectInterval string        `json:"connect_interval"`
	ConnectInterval    time.Duration `json:"-"`
	RawConnectPeriod   string        `json:"connect_period"`
	ConnectPeriod      time.Duration `json:"-"`
	filename           string
}

// LoadConnCfg loads the client config file.  Errors are logged but not
// returned.  TODO: revisit this design decision.
func (c *ConnCfg) Load(cfgFile string) error {
	c.filename = cfgFile
	b, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return fmt.Errorf("read of config file failed: %s", err)
	}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return fmt.Errorf("error unmarshaling confg file %s: %s", cfgFile, err)
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

func (c *ConnCfg) Save() error {
	j, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return fmt.Errorf("fail: cfg save: %s", err)
	}
	f, err := os.OpenFile(c.filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640)
	if err != nil {
		return fmt.Errorf("fail: cfg save: %s", err)
	}
	defer f.Close()
	n, err := f.Write(j)
	if err != nil {
		return fmt.Errorf("fail: cfg save: %s", err)
	}
	if n != len(j) {
		return fmt.Errorf("fail: cfg save: short write: wrote %d of %d bytes", n, len(j))
	}
	return nil
}

// Serialize serializes the struct.  The flatbuffers definition for this
// struct is in clientconf.fbs and the resulting definition is in
// client/ClientConf.go
func (c *Cfg) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	CfgStart(bldr)
	CfgAddHealthbeatInterval(bldr, c.HealthbeatInterval())
	CfgAddHealthbeatPushPeriod(bldr, c.HealthbeatPushPeriod())
	CfgAddPingPeriod(bldr, c.PingPeriod())
	CfgAddPongWait(bldr, c.PongWait())
	CfgAddSaveInterval(bldr, c.SaveInterval())
	bldr.Finish(CfgEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Deserialize deserializes the bytes into the current Cfg.
func (c *Cfg) Deserialize(p []byte) {
	c = GetRootAsCfg(p, 0)
}

// LoadInf loads the client.Inf from the received file.  If it doesn't exist
// a basic inf with its ID set to 0 is returned.
func LoadInf(fname string) *Inf {
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		bldr := flatbuffers.NewBuilder(0)
		InfStart(bldr)
		InfAddID(bldr, 0)
		bldr.Finish(InfEnd(bldr))
		return GetRootAsInf(bldr.Bytes[bldr.Head():], 0)
	}
	return GetRootAsInf(b, 0)
}

// Serialize serializes the Inf using flatbuffers and returns the []byte.
func (i *Inf) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	h := bldr.CreateByteString(i.Hostname())
	r := bldr.CreateByteString(i.Region())
	z := bldr.CreateByteString(i.Zone())
	d := bldr.CreateByteString(i.DataCenter())
	InfStart(bldr)
	InfAddID(bldr, i.ID())
	InfAddHostname(bldr, h)
	InfAddRegion(bldr, r)
	InfAddZone(bldr, z)
	InfAddDataCenter(bldr, d)
	bldr.Finish(InfEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Save the current Inf to a file.
func (i *Inf) Save(fname string) error {
	return ioutil.WriteFile(fname, i.Serialize(), 0600)
}
