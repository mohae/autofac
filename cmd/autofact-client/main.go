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
	addr            = flag.String("addr", "127.0.0.1:8675", "")
	reconnectPeriod = flag.String("reconnectperiod", "5m", "the amount of time to try and reconnect before quiting")
)

var cfg Cfg

// Cfg holds the client cfg
// TODO: implement
type Cfg struct {
	ID   uint32 `json:"id"`
	Addr string `json:"addr"`
}

func main() {
	os.Exit(realMain())
}

func realMain() int {
	flag.Parse()
	var err error
	reconnectPeriod, err := time.ParseDuration(*reconnectPeriod)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse of reconnect period failed: %s\n", err)
		return 1
	}

	// get a client
	c := autofact.NewClient(cfg.ID)
	// connect to the Server
	c.ServerURL = url.URL{Scheme: "ws", Host: *addr, Path: "/client"}
	c.ReconnectPeriod = reconnectPeriod
	// doneCh is used to signal that the connection has been closed
	doneCh := make(chan struct{})
	d, _ := time.ParseDuration("6s")
	c.HealthBeatPeriod = d
	d, _ = time.ParseDuration("30s")
	c.PushPeriod = d
	// must have a connection before doing anything
	for i := 0; i < 3; i++ {
		connected := c.Connect()
		if connected {
			break
		}
		// retry on fail until retry attempts have been exceeded
	}
	// start the healthbeat monitoring
	go c.HealthBeatFB()
	c.WS.SetPongHandler(c.PongHandler)
	c.WS.SetPingHandler(c.PingHandler)
	// start the connection handler
	go c.Listen(doneCh)
	go c.MessageWriter(doneCh)
	<-doneCh
	return 0
}
