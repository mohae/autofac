package autofact

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type ClientCfg struct {
	ID                      uint32 `json:"id"`
	ServerAddr              string `json:"server_address"`
	ServerPort              string `json:"server_port"`
	ServerID                uint32 `json:"server_id"`
	RawConnectInterval      string `json:"connect_interval"`
	ConnectInterval         time.Duration
	RawConnectPeriod        string `json:"connect_period"`
	ConnectPeriod           time.Duration
	RawHealthbeatInterval   string `json:"healthbeat_interval"`
	HealthbeatInterval      time.Duration
	RawHealthbeatPushPeriod string `json:"healthbeat_push_period"`
	HealthbeatPushPeriod    time.Duration
	PingPeriod              time.Duration `json:"-"`
	PongWait                time.Duration `json:"-"`
	RawSaveInterval         string        `json:"save_interval"`
	SaveInterval            time.Duration
	WriteWait               time.Duration `json:"-"`
	filename                string
}

// LoadClientCfg loads the client config file.  Errors are logged but not
// returned.  TODO: revisit this design decision.
func LoadClientCfg(cfgFile string) ClientCfg {
	var cfg ClientCfg
	cfg.filename = cfgFile
	b, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read of config file failed: %s", err)
		return cfg
	}
	err = json.Unmarshal(b, &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error unmarshaling confg file %s: %s", cfgFile, err)
	}
	cfg.ConnectInterval, err = time.ParseDuration(cfg.RawConnectInterval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing connect interval: %s", err)
	}
	cfg.ConnectPeriod, err = time.ParseDuration(cfg.RawConnectPeriod)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing connect period: %s", err)
	}
	cfg.HealthbeatInterval, err = time.ParseDuration(cfg.RawHealthbeatInterval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing healthbeat interval: %s", err)
	}
	cfg.HealthbeatPushPeriod, err = time.ParseDuration(cfg.RawHealthbeatPushPeriod)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing healthbeat push period %s", err)
	}
	cfg.SaveInterval, err = time.ParseDuration(cfg.RawSaveInterval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing save interval %s", err)
	}
	cfg.PingPeriod = PingPeriod
	cfg.PongWait = PongWait
	cfg.WriteWait = WriteWait
	return cfg
}

func (c *ClientCfg) Save() error {
	j, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return fmt.Errorf("fail: client cfg save: %s", err)
	}
	f, err := os.OpenFile(c.filename, os.O_CREATE|os.O_TRUNC|os.O_RDONLY, 0640)
	if err != nil {
		return fmt.Errorf("fail: client cfg save: %s", err)
	}
	defer f.Close()
	n, err := f.Write(j)
	if err != nil {
		return fmt.Errorf("fail: client cfg save: %s", err)
	}
	if n != len(j) {
		return fmt.Errorf("fail: client cfg save: short write: wrote %d of %d bytes", n, len(j))
	}
	return nil
}
