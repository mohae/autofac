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

var reconnectDuration time.Duration
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
	reconnectDuration, err = time.ParseDuration(*reconnectPeriod)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse of reconnect period failed: %s\n", err)
		return 1
	}

	// get a client
	c := autofact.NewClient(cfg.ID)
	// connect to the Server
	c.ServerURL = url.URL{Scheme: "ws", Host: *addr, Path: "/client"}
	// doneCh is used to signal that the connection has been closed
	doneCh := make(chan struct{})

	start := time.Now()
	retryEnd := start.Add(reconnectDuration)
	// connect to server; retry until the retry period has expired
	for {
		if time.Now().After(retryEnd) {
			fmt.Fprintln(os.Stderr, "timed out while trying to connect to the server")
			close(doneCh)
			return 1
		}
		err = c.DialServer()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
		fmt.Println("unable to connect to the server: retrying...")
	}
	d, _ := time.ParseDuration("6s")
	c.HealthBeatPeriod = d
	d, _ = time.ParseDuration("30s")
	c.PushPeriod = d
	// start the connection handler
	go c.HealthBeat()
	go connHandler(c, doneCh)
	// block until the done signal is set
	<-doneCh

	return 0
}
