package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/mohae/autofact"
	"github.com/mohae/autofact/cfg"
	"github.com/mohae/autofact/client"
)

var cfgFile = "autofact-client.json"
var infFile = "autoinf.dat"

// This is the default directory for autofact-client app data.
var defaultAutoFactDir = "$HOME/.autofact"
var bDBFile = "autofact.bdb" // bolt database file

// default
var connCfg cfg.Conn

// TODO: reconcile these flags with config file usage.  Probably add contour
// to handle this after the next refactor of contour.
func init() {
	flag.StringVar(&connCfg.ServerAddress, "address", "127.0.0.1", "the server address")
	flag.StringVar(&connCfg.ServerAddress, "a", "127.0.0.1", "the server address (short)")
	flag.StringVar(&connCfg.ServerPort, "port", "8675", "the connection port")
	flag.StringVar(&connCfg.ServerPort, "p", "8675", "the connection port (short)")
	connCfg.ConnectInterval = time.Duration(5) * time.Second
	connCfg.ConnectPeriod = time.Duration(15) * time.Minute
}

func main() {
	os.Exit(realMain())
}

func realMain() int {
	// Load the AUTOPATH value
	autopath := os.Getenv(autofact.PathVarName)
	if autopath == "" {
		autopath = defaultAutoFactDir
	}
	autopath = os.ExpandEnv(autopath)
	// make sure the autopath exists (create if it doesn't)
	err := os.MkdirAll(autopath, 0760)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create Autopath dir: %s\n", err)
		return 1
	}
	cfgFile = filepath.Join(autopath, cfgFile)
	infFile = filepath.Join(autopath, infFile)
	bDBFile = filepath.Join(autopath, bDBFile)
	// Load the client's information; if it can't be found or doesn't exist, e.g.
	// is a new client, a serialized client.Inf is returned with the client id set
	// to 0.  The server will provide the information.  The server also provides
	// updated client settings.
	// TODO: work out client inf setting management better.
	inf, err := cfg.LoadSysInf(infFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	// get a client
	c := client.New(inf)
	c.AutoPath = autopath
	err = c.Conn.Load(cfgFile)
	if err != nil {
		// If there was an error, not it and use the default settings
		fmt.Fprintf(os.Stderr, "using default settings: connection cfg: %s\n", err)
		c.Conn = connCfg
		c.Conn.SetFilename(cfgFile)
	}

	// open the database file
	err = c.DB.Open(bDBFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %s\n", err)
		return 1
	}
	defer c.DB.DB.Close()

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
	err = c.SysInf.Save(infFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "save client inf failed: %s\n", err)
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
	err = c.Conn.Save()
	if err != nil {
		fmt.Fprintf(os.Stderr, "save of cfg failed: %s\n", err)
	}
	<-doneCh
	return 0
}
