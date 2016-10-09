package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohae/autofact/conf"
	czap "github.com/mohae/zap"
	"github.com/uber-go/zap"
)

const (
	addressVar = "address"
	aVar       = "a"
	portVar    = "port"
	pVar       = "p"
)

var (
	connFile = "autofact.json"
	// This is the default directory for autofact-client app data.
	autofactPath    = "$HOME/.autofact"
	autofactEnvName = "AUTOFACT_PATH"
	// default
	connConf   conf.Conn
	serverless bool // if the client is being run without a server
)

// TODO determine loglevel mapping to actual usage:
// Proposed:
//  DebugLevel == not used
//	InfoLevel == Gathered data
//  WarnLevel == Connection info an non-error messages: status type
//  ErrorLevel == Errors
//  PanicLevel == Panic: shouldn't be used
//  FatalLevel == Unrecoverable error that results in app shutdown
// TODO: implement data logging
var (
	log      zap.Logger
	loglevel = zap.LevelFlag("loglevel", zap.WarnLevel, "log level")
	logfile  string
	f        *os.File

	dataLog     czap.Logger // use mohae's fork to support level description override
	dataLogFile string
	dataFile    *os.File
	// These are globals so that they can be closed

	isStdErr bool
)

// TODO: reconcile these flags with config file usage.  Probably add contour
// to handle this after the next refactor of contour.
// TODO: make connectInterval/period handling consistent, e.g. should they be
// flags, what is precedence in relation to Conn?
func init() {
	flag.StringVar(&connConf.ServerAddress, addressVar, "127.0.0.1", "the server address")
	flag.StringVar(&connConf.ServerAddress, aVar, "127.0.0.1", "the server address (short)")
	flag.StringVar(&connConf.ServerPort, portVar, "8675", "the connection port")
	flag.StringVar(&connConf.ServerPort, pVar, "8675", "the connection port (short)")
	flag.StringVar(&logfile, "logfile", "autofact.log", "application log file; if empty stderr will be used")
	flag.StringVar(&logfile, "l", "autofact.log", "application log file; if empty stderr will be used")
	flag.StringVar(&dataLogFile, "datafile", "autodata.json", "the file that the collected data gets written when running serverless")
	flag.StringVar(&dataLogFile, "d", "autodata.json", "the file that the collected data gets written when running serverless")
	flag.BoolVar(&serverless, "serverless", false, "serverless: the client will run standalone and write the collected data to the log")
	connConf.ConnectInterval.Duration = 5 * time.Second
	connConf.ConnectPeriod.Duration = 15 * time.Minute

	// override czap description for InfoLevel
	czap.InfoString = "data"
}

func main() {
	// Load the AUTOPATH value
	tmp := os.Getenv(autofactEnvName)
	if tmp != "" {
		autofactPath = tmp
	}
	autofactPath = os.ExpandEnv(autofactPath)

	// make sure the autofact path exists (create if it doesn't)
	err := os.MkdirAll(autofactPath, 0760)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create AUTOFACT_PATH: %s\n", err)
		fmt.Fprintln(os.Stderr, "startup error: exiting")
		os.Exit(1)
	}

	// finalize the paths
	connFile = filepath.Join(autofactPath, connFile)

	// process the settings
	var connMsg string
	err = connConf.Load(connFile)
	if err != nil {
		// capture the error for logging once it is setup and continue.  An error
		// is not a show stopper as the file may not exist if this is the first
		// time autofact has run on this node.
		connMsg = fmt.Sprintf("using default settings")
	}

	// Parse the flags.
	flag.Parse()

	// now that everything is parsed; set up logging
	SetLogging()
	defer CloseLog()
	// if there was an error reading the connection configuration and this isn't
	// being run serverless, log it
	if connMsg != "" && !serverless {
		log.Warn(
			err.Error(),
			zap.String("conf", connMsg),
		)
	}

	// TODO add env var support

	// get a client
	c := NewClient(connConf)
	c.AutoPath = autofactPath

	// handle signals
	go handleSignals(c)

	// doneCh is used to signal that the connection has been closed
	doneCh := make(chan struct{})

	if serverless { // open the datafile to use
		if dataLogFile == "" {
			log.Fatal(
				"filename is not specified",
				zap.String("op", "open data file"),
			)
		}
		SetDataLog()
		defer dataFile.Close()
	} else { // connect to the server
		// connect to the Server
		c.ServerURL = url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%s", c.ServerAddress, c.ServerPort), Path: "/client"}

		// must have a connection before doing anything
		for i := 0; i < 3; i++ {
			connected := c.Connect()
			if connected {
				break
			}
			// retry on fail until retry attempts have been exceeded
		}
		if !c.IsConnected() {
			CloseLog() // defer doesn't run on fatal
			log.Fatal(
				"unable to connect",
				zap.String("server", c.ServerURL.String()),
			)
		}
	}

	// start the go routines for socket communications
	go c.Listen(doneCh)
	go c.MemInfo(doneCh)
	go c.CPUUtilization(doneCh)
	go c.NetUsage(doneCh)
	// start the connection handler
	go c.MessageWriter(doneCh)

	if !serverless {
		// if connected, save the conf: this will also save the ClientID
		err = c.Conn.Save()
		if err != nil {
			log.Error(
				err.Error(),
				zap.String("op", "save conn"),
				zap.String("file", c.Filename),
			)
		}
	}
	<-doneCh
}

func SetLogging() {
	// if logfile is empty, use Stderr
	var err error
	if logfile == "" {
		f = os.Stderr
		isStdErr = true
	} else {
		f, err = os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
		if err != nil {
			panic(err)
		}
	}
	log = zap.New(
		zap.NewJSONEncoder(
			zap.RFC3339Formatter("ts"),
		),
		zap.Output(f),
	)
	log.SetLevel(*loglevel)
}

func SetDataLog() {
	var err error
	dataFile, err = os.OpenFile(dataLogFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
	if err != nil {
		log.Fatal(
			err.Error(),
			zap.String("op", "open datafile"),
			zap.String("filename", dataLogFile),
		)
	}
	dataLog = czap.New(
		czap.NewJSONEncoder(
			czap.RFC3339Formatter("ts"),
		),
		czap.Output(dataFile),
	)
	dataLog.SetLevel(czap.InfoLevel)
}

// CloseLog closes the log file before exiting.
func CloseLog() {
	if !isStdErr && f != nil {
		f.Close()
	}
	// If running serverless, close the data file.
	if serverless {
		dataFile.Close()
	}
}

func handleSignals(c *Client) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	v := <-ch
	log.Info(
		"os signal received: shutting down autofact",
		zap.Object("signal", v.String()),
	)
	// If there's a connection send a close signal
	if c.IsConnected() {
		log.Debug(
			"closing connection",
			zap.String("op", "shutdown"),
		)
		c.WS.WriteMessage(websocket.CloseMessage, []byte(string(c.Conn.ID)+" shutting down"))
	}
	CloseLog()

	os.Exit(1)
}
