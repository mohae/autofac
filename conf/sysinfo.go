package conf

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/google/flatbuffers/go"
	"github.com/mohae/autofact/util"
)

// SysInfo holds the configuration information for what system information
// should be collected on either node startup, serverless, or when a node
// connects to the server.
var SysInfFile = "autoinf.json"
var GetSysInfo bool

func init() {
	flag.StringVar(&SysInfFile, "sysinfconf", SysInfFile, "")
}

type SysInfo struct {
	CPU      bool   `json:"cpu"`       // Collect basic CPU information.
	CPUFlags bool   `json:"cpu_flags"` // Collect the CPU Flags.
	Mem      bool   `json:"mem"`       // Collect information about the systems memory.
	NetInf   bool   `json:"net_inf"`   // Collect information about the system's network interfaces.
	Filename string `json:"-"`
}

// Load loads the Collect configuration from the specified file.
func (s *SysInfo) Load(fname string) error {
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, s)
	if err != nil {
		return fmt.Errorf("%s unmarshal error: %s", fname, err)
	}
	return nil
}

func (s *SysInfo) SaveJSON() error {
	b, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return fmt.Errorf("error marshaling sysinfo conf to json: %s", err)
	}
	err = ioutil.WriteFile(s.Filename, b, 0600)
	if err != nil {
		return fmt.Errorf("%s save error: %s", s.Filename, err)
	}
	return nil
}

func (s *SysInfo) UseDefaults() {
	s.CPU = true
	s.CPUFlags = true
	s.Mem = true
	s.NetInf = true
}

// Serialize serializes Sysinfo as SysInf using flatbuffers.
func (s *SysInfo) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	SysInfStart(bldr)
	SysInfAddCPU(bldr, util.BoolToByte(s.CPU))
	SysInfAddCPUFlags(bldr, util.BoolToByte(s.CPUFlags))
	SysInfAddMem(bldr, util.BoolToByte(s.Mem))
	SysInfAddNetInf(bldr, util.BoolToByte(s.NetInf))
	bldr.Finish(SysInfEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// Deserialize deserializes flatbuffer serializes bytes into SysInfo.
func (s *SysInfo) Deserialize(p []byte) {
	inf := GetRootAsSysInf(p, 0)
	s.CPU = util.ByteToBool(inf.CPU())
	s.CPUFlags = util.ByteToBool(inf.CPUFlags())
	s.Mem = util.ByteToBool(inf.Mem())
	s.NetInf = util.ByteToBool(inf.NetInf())
}
