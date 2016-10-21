package conf

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/mohae/autofact/util"
)

// Defaults for Client Conf: if file doesn't exist.
var (
	// Pull
	DefaultHealthbeatPeriod = util.Duration{10 * time.Second}
	// Client Side
	DefaultMemInfoPeriod        = util.Duration{15 * time.Second}
	DefaultCPUUtilizationPeriod = util.Duration{15 * time.Second}
	DefaultNetUsagePeriod       = util.Duration{15 * time.Second}
)

// Conf is used to hold flag arguments passed on start
type Conf struct {
	args []*flag.Flag // all flags that were visited
}

// Visited builds a list of the names of the flags that were passed as args.
func (c *Conf) Visited(a *flag.Flag) {
	c.args = append(c.args, a)
}

// Args returns all of the args that were passed.
func (c *Conf) Args() []*flag.Flag {
	return c.args
}

// Flag returns the requested flag, if it was set, or nil.
func (c *Conf) Flag(s string) *flag.Flag {
	for _, v := range c.args {
		if s == v.Name {
			return v
		}
	}
	return nil
}

// Conn holds the connection information for a node.  This is all that is
// persisted on a client node.
type Conn struct {
	ID              []byte        `json:"id"`
	ServerAddress   string        `json:"server_address"`
	ServerPort      string        `json:"server_port"`
	ServerID        uint32        `json:"server_id"`
	ConnectInterval util.Duration `json:"connect_interval"`
	ConnectPeriod   util.Duration `json:"connect_period"`
	Filename        string        `json:"-"`
	Conf            `json:"-"`
}

// LoadConn loads the config file.  The Conn's filename is set during this
// operation.
// TODO: revisit this design decision.
func (c *Conn) Load(name string) error {
	c.Filename = name
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
	f, err := os.OpenFile(c.Filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640)
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

// Serialize serializes the Client conf.
func (c *Client) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	id := bldr.CreateByteVector(c.IDBytes())
	ClientStart(bldr)
	ClientAddID(bldr, id)
	ClientAddHealthbeatPeriod(bldr, c.HealthbeatPeriod())
	ClientAddMemInfoPeriod(bldr, c.MemInfoPeriod())
	ClientAddCPUUtilizationPeriod(bldr, c.CPUUtilizationPeriod())
	ClientAddNetUsagePeriod(bldr, c.NetUsagePeriod())
	bldr.Finish(ClientEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Deserialize deserializes the bytes into the current Client Conf.
func (c *Client) Deserialize(p []byte) {
	c = GetRootAsClient(p, 0)
}

// Collect defines the collection periods of various data.
type Collect struct {
	HealthbeatPeriod     util.Duration `json:"healthbeat_period"`
	CPUUtilizationPeriod util.Duration `json:"cpuutilization_period"`
	MemInfoPeriod        util.Duration `json:"meminfo_period"`
	NetUsagePeriod       util.Duration `json:"netusage_period"`
	Filename             string        `json:"-"`
}

// Load loads the Collect configuration from the specified file.
func (c *Collect) Load(dir, name string) error {
	c.Filename = name
	b, err := ioutil.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, c)
	if err != nil {
		return fmt.Errorf("%s unmarshal error: %s", filepath.Join(dir, name), err)
	}
	return nil
}

// Returns a Collect with application defaults.  This is called when the
// collect file cannot be found.
func (c *Collect) UseDefaults() {
	c.HealthbeatPeriod = DefaultHealthbeatPeriod
	c.CPUUtilizationPeriod = DefaultCPUUtilizationPeriod
	c.MemInfoPeriod = DefaultMemInfoPeriod
	c.NetUsagePeriod = DefaultNetUsagePeriod
}

func (c *Collect) SaveJSON(dir string) error {
	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return fmt.Errorf("%s marshal error: %s", c.Filename, err)
	}
	err = ioutil.WriteFile(filepath.Join(dir, c.Filename), b, 0600)
	if err != nil {
		return fmt.Errorf("%s save error: %s", filepath.Join(dir, c.Filename), err)
	}
	return nil
}

// Serialize serializes the struct as a conf.Client.
func (c *Collect) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	ClientStart(bldr)
	ClientAddMemInfoPeriod(bldr, c.MemInfoPeriod.Int64())
	ClientAddCPUUtilizationPeriod(bldr, c.CPUUtilizationPeriod.Int64())
	ClientAddNetUsagePeriod(bldr, c.NetUsagePeriod.Int64())
	bldr.Finish(ClientEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Deserialize deserializes serialized conf.Client into Collect.
func (c *Collect) Deserialize(p []byte) {
	cnf := GetRootAsClient(p, 0)
	c.MemInfoPeriod.Set(cnf.MemInfoPeriod())
	c.CPUUtilizationPeriod.Set(cnf.CPUUtilizationPeriod())
	c.NetUsagePeriod.Set(cnf.NetUsagePeriod())
}
