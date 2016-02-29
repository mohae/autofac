package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/mohae/autofac"
)

// flags
var (
	addr = flag.String("addr", "127.0.0.1:8675", "")
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

	// get a client
	c := autofac.NewClient(cfg.ID)
	// connect to the Server
	c.ServerURL = url.URL{Scheme: "ws", Host: *addr, Path: "/client"}
	// doneCh is used to signal that the connection has been closed
	doneCh := make(chan struct{})

	// connect to server
	err := c.DialServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to %s: %s\n", c.ServerURL.String(), err)
		return 1
	}

	// start the connection handler
	go connHandler(c, doneCh)

	// block until the done signal is set
	<-doneCh

	return 0
}
