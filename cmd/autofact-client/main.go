package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/mohae/autofact"
)

// flags
var (
	addr                 string
	port                 string
	healthbeatInterval   string
	healthbeatPushPeriod string
	connectInterval      string
	connectPeriod        string
)

var cfg autofact.ClientCfg
var cfgFile = "autofact-client.json"

// initialize the cfg to defaults
func init() {
	cfg.ServerAddr = "127.0.0.1"
	cfg.ServerPort = "8675"
	cfg.HealthbeatInterval = time.Duration(5) * time.Second
	cfg.HealthbeatPushPeriod = time.Duration(10) * time.Second
	cfg.ConnectInterval = time.Duration(5) * time.Second
	cfg.ConnectPeriod = time.Duration(15) * time.Minute
	flag.StringVar(&addr, "addr", "", "address of server")
	flag.StringVar(&addr, "a", "", "address of server (short)")
	flag.StringVar(&healthbeatInterval, "healthbeatinterval", "", "the interval between gathering basic system stats")
	flag.StringVar(&healthbeatInterval, "i", "", "the interval between gathering basic system stats (short)")
	flag.StringVar(&healthbeatPushPeriod, "healthbeatpushperiod", "", "the amount of time before pushing healthbeat stats")
	flag.StringVar(&healthbeatPushPeriod, "h", "", "the amount of time before pushing healthbeat stats (short)")
	flag.StringVar(&port, "port", "", "port to use for websockets")
	flag.StringVar(&port, "p", "", "port to use for websockets (short)")
	flag.StringVar(&connectInterval, "connectinterval", "", "the interval between attempting to connect to the server")
	flag.StringVar(&connectInterval, "c", "", "the interval between attempting to connect to the server (short)")
	flag.StringVar(&connectPeriod, "connectperiod", "", "the amount of time to try and connect before quitting")
	flag.StringVar(&connectPeriod, "n", "", "the amount of time to try and connect before quitting (short)")
}

func main() {
	os.Exit(realMain())
}

func realMain() int {
	// TODO: the way app settings are handled are a bit wonky.  Should probably
	// use contour or refactor this.

	// errors are logged but don't stop execution
	cfg = autofact.LoadClientCfg(cfgFile)
	flag.Parse()

	if addr != "" {
		cfg.ServerAddr = addr
	}
	if healthbeatInterval != "" {
		d, err := time.ParseDuration(healthbeatInterval)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse of healthbeat period failed: %s\n", err)
			return 1
		}
		cfg.HealthbeatInterval = d
	}
	if healthbeatPushPeriod != "" {
		d, err := time.ParseDuration(healthbeatPushPeriod)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse of healthbeat push interval failed: %s\n", err)
			return 1
		}
		cfg.HealthbeatPushPeriod = d
	}
	if port != "" {
		cfg.ServerPort = port
	}
	if connectPeriod != "" {
		d, err := time.ParseDuration(connectPeriod)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse of reconnect period failed: %s\n", err)
			return 1
		}
		cfg.ConnectPeriod = d
	}

	// get a client
	c := autofact.NewClient(uint32(0))
	c.Cfg = cfg
	// connect to the Server
	c.ServerURL = url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%s", cfg.ServerAddr, cfg.ServerPort), Path: "/client"}
	// doneCh is used to signal that the connection has been closed
	doneCh := make(chan struct{})
	// must have a connection before doing anything
	for i := 0; i < 3; i++ {
		connected := c.Connect()
		if connected {
			break
		}
		// retry on fail until retry attempts have been exceeded
	}
	if !c.IsConnected() {
		fmt.Fprintf(os.Stderr, "unable to connect to %s\n", c.ServerURL.String())
		return 1
	}
	// start the healthbeat monitoring
	go c.Healthbeat()
	c.WS.SetPongHandler(c.PongHandler)
	c.WS.SetPingHandler(c.PingHandler)
	// start the connection handler
	go c.Listen(doneCh)
	go c.MessageWriter(doneCh)
	<-doneCh
	return 0
}
