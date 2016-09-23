package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/mohae/autofact/cfg"
	"github.com/mohae/autofact/client"
)

var (
	connFile = "autofact.json"
	infFile  = "autofact.dat" // contains the node id and other info as serialized data
	// This is the default directory for autofact-client app data.
	autofactPath    = "$HOME/.autofact"
	autofactEnvName = "AUTOFACT_PATH"
	// default
	connCfg    cfg.Conn
	addressVar = "address"
	aVar       = "a"
	portVar    = "port"
	pVar       = "p"
)

// TODO: reconcile these flags with config file usage.  Probably add contour
// to handle this after the next refactor of contour.
// TODO: make connectInterval/period handling consistent, e.g. should they be
// flags, what is precedence in relation to Conn?
func init() {
	flag.StringVar(&connCfg.ServerAddress, addressVar, "127.0.0.1", "the server address")
	flag.StringVar(&connCfg.ServerAddress, aVar, "127.0.0.1", "the server address (short)")
	flag.StringVar(&connCfg.ServerPort, portVar, "8675", "the connection port")
	flag.StringVar(&connCfg.ServerPort, pVar, "8675", "the connection port (short)")
	connCfg.ConnectInterval = 5 * time.Second
	connCfg.ConnectPeriod = 15 * time.Minute
}

func main() {
	os.Exit(realMain())
}

func realMain() int {
	// Load the AUTOPATH value
	tmp := os.Getenv(autofactEnvName)
	if tmp != "" {
		autofactPath = tmp
	}
	autofactPath = os.ExpandEnv(autofactPath)

	// make sure the autofact path exists (create if it doesn't)
	err := os.MkdirAll(autofactPath, 0760)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create AUTOFACT_PATH dir: %s\n", err)
		return 1
	}

	// finalize the paths
	connFile = filepath.Join(autofactPath, connFile)
	infFile = filepath.Join(autofactPath, infFile)

	// process the settings
	err = connCfg.Load(connFile)
	if err != nil {
		// Log the error and continue.  An error is not a show stopper as the file
		// may not exist if this is the first time autofact has run on this node.
		fmt.Fprintf(os.Stderr, "using default settings: connection cfg: %s\n", err)
	}

	// Parse the flags.
	flag.Parse()

	// TODO add env var support

	// Load the client's information; if it can't be found or doesn't exist, e.g.
	// is a new client, a serialized client.Inf is returned with the client id set
	// to 0.  The server will provide the information.  The server also provides
	// updated client settings.
	// TODO: elide this; the info should come from the server
	inf, err := cfg.LoadNodeInf(infFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading connection information from file: %s", err)
	}

	// get a client
	c := client.New(inf)
	c.AutoPath = autofactPath
	c.Conn = connCfg

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
	err = c.NodeInf.Save(infFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "save client inf failed: %s\n", err)
	}
	// start the go routines first
	go c.Listen(doneCh)
	go c.Healthbeat()
	// start the connection handler
	go c.MessageWriter(doneCh)
	// if connected, save the cfg: this will also save the ClientID
	err = c.Conn.Save()
	if err != nil {
		fmt.Fprintf(os.Stderr, "save of cfg failed: %s\n", err)
	}
	<-doneCh
	return 0
}
