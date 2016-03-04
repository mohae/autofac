package autofact

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type ClientCfg struct {
	Addr                    string `json:"address"`
	Port                    string `json:"port"`
	RawConnectInterval      string `json:"connect_interval"`
	ConnectInterval         time.Duration
	RawConnectPeriod        string `json:"connect_period"`
	ConnectPeriod           time.Duration
	RawHealthbeatInterval   string `json:"healthbeat_interval"`
	HealthbeatInterval      time.Duration
	RawHealthbeatPushPeriod string `json:"healthbeat_push_period"`
	HealthbeatPushPeriod    time.Duration
}

// LoadClientCfg loads the client config file.  Errors are logged but not
// returned.  TODO: revisit this design decision.
func LoadClientCfg(cfgFile string) ClientCfg {
	var cfg ClientCfg
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
	return cfg
}
