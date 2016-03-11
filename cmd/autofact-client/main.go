package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/mohae/autofact/client"
)

var cfgFile = "autofact-client.json"
var infFile = "autoinf.dat"

// default
var connCfg client.ConnCfg

func init() {
	flag.StringVar(&connCfg.ServerAddress, "address", "127.0.0.1", "the server address")
	flag.StringVar(&connCfg.ServerAddress, "a", "127.0.0.1", "the server address (short)")
	flag.StringVar(&connCfg.ServerPort, "port", "8086", "the connection port")
	flag.StringVar(&connCfg.ServerPort, "p", "8086", "the connection port (short)")
	connCfg.ConnectInterval = time.Duration(5) * time.Second
	connCfg.ConnectPeriod = time.Duration(15) * time.Minute
}

func main() {
	os.Exit(realMain())
}

func realMain() int {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to get hostname: %s", err)
		return 1
	}

	inf := client.LoadInf(infFile)
	// get a client
	c := client.New(hostname, inf)
	err = c.ConnCfg.Load(cfgFile)
	if err != nil {
		// If there was an error, not it and use the default settings
		fmt.Fprintf(os.Stderr, "using default settings: connection cfg error: %s", err)
		c.ConnCfg = connCfg
	}

	// connect to the Server
	c.ServerURL = url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%s", c.ServerAddress, c.ServerPort), Path: "/client"}
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
	// save the client inf
	err = c.Inf.Save(infFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "save client inf: %s", err)
	}
	// start the go routines first
	go c.Listen(doneCh)
	go c.Healthbeat()
	// start the healthbeat monitoring
	c.WS.SetPongHandler(c.PongHandler)
	c.WS.SetPingHandler(c.PingHandler)
	// start the connection handler
	go c.MessageWriter(doneCh)
	// if connected, save the cfg: this will also save the ClientID
	err = c.ConnCfg.Save()
	if err != nil {
		fmt.Fprintf(os.Stderr, "save of cfg failed: %s\n", err)
	}
	<-doneCh
	return 0
}
