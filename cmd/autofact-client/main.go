package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/mohae/autofact/client"
)

var cfgFile = "autofact-client.json"

func main() {
	os.Exit(realMain())
}

func realMain() int {
	// get a client
	c := client.New(uint32(0))
	err := c.ConnCfg.Load(cfgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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
	// if connected, save the cfg: this will also save the ClientID
	err = c.ConnCfg.Save()
	if err != nil {
		fmt.Fprintf(os.Stderr, "save of cfg failed: %s\n", err)
	}
	go c.Listen(doneCh)
	// start the healthbeat monitoring
	go c.Healthbeat()
	c.WS.SetPongHandler(c.PongHandler)
	c.WS.SetPingHandler(c.PingHandler)
	// start the connection handler
	go c.MessageWriter(doneCh)
	<-doneCh
	return 0
}
