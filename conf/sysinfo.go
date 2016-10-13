package conf

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
)

// SysInfo holds the configuration information for what system information
// should be collected on either node startup, serverless, or when a node
// connects to the server.
var SysInfFile = "autoinf.json"
var SysInf bool

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
