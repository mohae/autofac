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
	connFile    = "autofact.json"
	collectFile = "autocollect.json"
	// This is the default directory for autofact-client app data.
	autofactPath    = "$HOME/.autofact"
	autofactEnvName = "AUTOFACT_PATH"
	// configuration info
	connConf conf.Conn

	// client configuration: used for serverless

	serverless bool
)

// Vars for logging and local data output, if applicable.
var (
	log      zap.Logger // application log
	loglevel = zap.LevelFlag("loglevel", zap.WarnLevel, "log level")
	logOut   string
	logFile  *os.File

	data     czap.Logger // use mohae's fork to support level description override
	dataOut  string
	dataFile *os.File
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
	flag.StringVar(&logOut, "logout", "stderr", "log output; if empty stderr will be used")
	flag.StringVar(&logOut, "l", "stderr", "log output; if empty stderr will be used")
	flag.StringVar(&dataOut, "dataout", "stdout", "serverless mode data output, if empty stderr will be used")
	flag.StringVar(&dataOut, "d", "stdout", "serverless mode data output, if empty stderr will be used")
	flag.BoolVar(&serverless, "serverless", false, "serverless: the client will run standalone and write the collected data to the log")
	connConf.ConnectInterval.Duration = 5 * time.Second
	connConf.ConnectPeriod.Duration = 15 * time.Minute

	// override czap description for InfoLevel
	czap.WarnString = "data"
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

	// process the settings: this gets read first just in case flags override
	var connMsg string
	err = connConf.Load(connFile)
	if err != nil {
		// capture the error for logging once it is setup and continue.  An error
		// is not a show stopper as the file may not exist if this is the first
		// time autofact has run on this node.
		connMsg = fmt.Sprintf("using default connection settings")
	}

	// Parse the flags.
	flag.Parse()

	// now that everything is parsed; set up logging
	SetLogging()
	defer CloseOut()
	// if there was an error reading the connection configuration and this isn't
	// being run serverless, log it
	if connMsg != "" && !serverless {
		log.Warn(
			err.Error(),
			zap.String("op", fmt.Sprintf("load %s", connFile)),
			zap.String("conf", connMsg),
		)
	}

	// TODO add env var support

	// get a client
	c := NewClient(connConf, collectFile)
	c.AutoPath = autofactPath

	// if serverless: load the collection configuration
	if serverless {
		collectFile = filepath.Join(autofactPath, collectFile)
		err = c.Collect.Load(collectFile)
		if err != nil {
			log.Warn(
				err.Error(),
				zap.String("op", fmt.Sprintf("load %s", collectFile)),
				zap.String("conf", "using default collect settings"),
			)
			c.Collect.UseDefaults()
			err = c.Collect.SaveJSON()
			if err != nil {
				log.Warn(
					err.Error(),
					zap.String("op", fmt.Sprintf("save %s", collectFile)),
				)
			}
		}
	}

	// handle signals
	go handleSignals(c)

	// doneCh is used to signal that the connection has been closed
	doneCh := make(chan struct{})

	// Set up the output destination.
	if serverless { // open the datafile to use
		SetDataOut()
		WriteSystemInfo()
	}

	if !serverless { // connect to the server
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
			log.Error(
				"unable to connect",
				zap.String("server", c.ServerURL.String()),
			)
			CloseOut() // defer doesn't run on fatal
			os.Exit(1)
		}
		// if connected, save the conf: this will also save the ClientID
		err = c.Conn.Save()
		if err != nil {
			log.Error(
				err.Error(),
				zap.String("op", "save conn"),
				zap.String("file", c.Conn.Filename),
			)
		}
	}

	// set up the data processing
	if serverless {
		// since there isn't a server pull for healthbeat, a local ticker is started
		go c.HealthbeatLocal(doneCh)
		c.CPUUtilization = c.CPUUtilizationLocal
		c.MemInfo = c.MemInfoLocal
		c.NetUsage = c.NetUsageLocal
	} else {
		// assign the
		c.LoadAvg = LoadAvgFB
		c.CPUUtilization = c.CPUUtilizationFB
		c.MemInfo = c.MemInfoFB
		c.NetUsage = c.NetUsageFB

		// start the listener
		go c.Listen(doneCh)
		// start the message writer
		go c.MessageWriter(doneCh)
	}

	go c.CPUUtilization(doneCh)
	go c.MemInfo(doneCh)
	go c.NetUsage(doneCh)

	<-doneCh
}

func SetLogging() {
	// if logfile is empty, use Stderr
	var err error
	if logOut == "" || logOut == "stderr" {
		logFile = os.Stderr
		goto newLog
	}
	if logOut == "stdout" {
		logFile = os.Stdout
		goto newLog
	}
	logFile, err = os.OpenFile(logOut, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
	if err != nil {
		panic(err)
	}
newLog:
	log = zap.New(
		zap.NewJSONEncoder(
			zap.RFC3339Formatter("ts"),
		),
		zap.Output(logFile),
	)
	log.SetLevel(*loglevel)
}

func SetDataOut() {
	var err error
	if dataOut == "" || dataOut == "stdout" {
		dataFile = os.Stdout
		goto newData
	}
	if dataOut == "stderr" {
		dataFile = os.Stderr
		goto newData
	}
	dataFile, err = os.OpenFile(dataOut, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
	if err != nil {
		log.Fatal(
			err.Error(),
			zap.String("op", "open datafile"),
			zap.String("filename", dataOut),
		)
	}
newData:
	data = czap.New(
		czap.NewJSONEncoder(
			czap.RFC3339Formatter("ts"),
		),
		czap.Output(dataFile),
	)
	data.SetLevel(czap.WarnLevel)
}

// CloseOut closes the local output destinations before shutdown.
func CloseOut() {
	if logFile != nil {
		logFile.Close()
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
	CloseOut()

	os.Exit(1)
}
