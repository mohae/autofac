package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/mohae/autofact/conf"
	"github.com/uber-go/zap"
)

const (
	addressVar      = "address"
	aVar            = "a"
	clientConfVar   = "clientconf"
	cVar            = "c"
	dbfileVar       = "dbfile"
	dVar            = "d"
	influxDBNameVar = "dbname"
	iVar            = "i"
	passwordVar     = "password"
	pVar            = "p"
	portVar         = "port"
	oVar            = "o"
	usernameVar     = "username"
	uVar            = "u"
	serverIDVar     = "serverid"
	sVar            = "s"
)

var (
	srvr     = newServer()
	connConf conf.Conn

	// Logging
	log      zap.Logger
	loglevel = zap.LevelFlag("loglevel", zap.WarnLevel, "log level")
	logDest  string
	logOut   *os.File

	// The default directory used by Autofactory for app data.
	autofactoryPath    = "$HOME/.autofactory"
	autofactoryEnvName = "AUTOFACTORY_PATH"

	serverID       string
	clientConfFile string
	influxUser     string
	influxPassword string
)

// flags
func init() {
	flag.StringVar(&connConf.ServerPort, portVar, "8675", "port to use for websockets")
	flag.StringVar(&connConf.ServerPort, oVar, "8675", "port to use for websockets (short)")
	flag.StringVar(&clientConfFile, clientConfVar, "autoclient.json", "location of client configuration file")
	flag.StringVar(&clientConfFile, cVar, "autoclient.json", "location of client configuration file (short)")
	flag.StringVar(&serverID, serverIDVar, "autosrvr", "ID of the autofactory server")
	flag.StringVar(&serverID, sVar, "autosrvr", "ID of the autofactory server")
	flag.StringVar(&srvr.BoltDBFile, dbfileVar, "autofactory.bdb", "location of the autofactory database file")
	flag.StringVar(&srvr.BoltDBFile, dVar, "autofactory.bdb", "location of the autfactory database file (short)")
	flag.StringVar(&srvr.InfluxDBName, influxDBNameVar, "autofacts", "name of the InfluxDB to connect to")
	flag.StringVar(&srvr.InfluxDBName, iVar, "autofacts", "name of the InfluxDB to connect to (short)")
	flag.StringVar(&srvr.InfluxAddress, addressVar, "127.0.0.1:8086", "the address of the InfluxDB")
	flag.StringVar(&srvr.InfluxAddress, aVar, "http://127.0.0.1:8086", "the address of the InfluxDB (short)")
	flag.StringVar(&influxUser, usernameVar, "autoadmin", "the username of the InfluxDB user")
	flag.StringVar(&influxUser, uVar, "autoadmin", "the username of the InfluxDB user (short)")
	flag.StringVar(&influxPassword, passwordVar, "thisisnotapassword", "the username of the InfluxDB user")
	flag.StringVar(&influxPassword, pVar, "thisisnotapassword", "the username of the InfluxDB user (short)")
	flag.StringVar(&logDest, "logdestination", "stderr", "log output destination; if empty stderr will be used")
	flag.StringVar(&logDest, "l", "stderr", "log output; if empty stderr will be used")
}

func main() {
	os.Exit(realMain())
}

// realMain is used to allow defers to run.
func realMain() int {
	// Load the AUTOFACTORY_PATH value
	tmp := os.Getenv(autofactoryEnvName)
	if tmp != "" {
		autofactoryPath = tmp
	}
	autofactoryPath = os.ExpandEnv(autofactoryPath)

	// make sure the autopath exists (create if it doesn't)
	err := os.MkdirAll(autofactoryPath, 0760)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create AUTOFACTORY_PATH: %s\n", err)
		fmt.Fprintln(os.Stderr, "startup error: exiting")
		return 1
	}

	srvr.BoltDBFile = filepath.Join(autofactoryPath, srvr.BoltDBFile)

	flag.Parse()

	// now that everything is parsed; set up logging
	SetLogging()
	defer CloseLog()

	srvr.ID = []byte(serverID)
	srvr.NewSnowflakeGenerator()
	srvr.AutoPath = autofactoryPath
	// load the default client conf; this is used for new clients.
	// TODO: in the future, there should be support for enabling setting per
	// client, or group, or role, or pod, etc.
	err = srvr.Collect.Load(srvr.AutoPath, clientConfFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error(
				err.Error(),
				zap.String("op", "read conf"),
				zap.String("file", clientConfFile),
			)
			return 1
		}
		// If it didn't exist; use application defaults
		log.Warn(
			"conf file not found, using default values for client configuration",
			zap.String("op", "read conf"),
			zap.String("file", clientConfFile),
		)
		// write this out to the app dir
		srvr.Collect.UseDefaults()
		fmt.Println(autofactoryPath, srvr.Collect.Filename)
		err = srvr.Collect.SaveJSON(srvr.AutoPath)
		if err != nil { // a save error isn't fatal
			log.Error(
				err.Error(),
				zap.String("op", "save conf"),
				zap.String("file", clientConfFile),
			)
		}
	}
	// bdb is used as the extension for bolt db.
	err = srvr.Bolt.Open(srvr.BoltDBFile)
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "open boltdb"),
			zap.String("file", srvr.BoltDBFile),
		)
		return 1
	}
	defer srvr.Bolt.Close()

	// connect to Influx
	// TODO make this optional; if Influx isn't going to be used, leverage
	// zap to write the data to a structured log.
	err = srvr.connectToInfluxDB(influxUser, influxPassword)
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "connect to influxdb"),
			zap.String("db", srvr.InfluxDBName),
		)
		return 1
	}
	go handleSignals(&srvr)
	// start the Influx writer
	// TODO: influx writer should handle done channel signaling
	go srvr.InfluxClient.Write()
	srvr.LoadInventory()
	http.HandleFunc("/client", serveClient)
	err = http.ListenAndServe(fmt.Sprintf(":%s", connConf.ServerPort), nil)
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "start server"),
			zap.String("port", connConf.ServerPort),
		)
		return 1
	}
	return 0
}

func handleSignals(srvr *server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	v := <-c
	log.Info(
		"os signal received: shutting down autofactory",
		zap.Object("signal", v),
	)
	srvr.Bolt.Close()
	CloseLog()
	os.Exit(1)
}

func SetLogging() {
	// if logfile is empty, use Stderr
	var err error
	if logDest == "" || logDest == "stderr" {
		logOut = os.Stderr
		goto newLog
	}
	if logDest == "stdout" {
		logOut = os.Stdout
		goto newLog
	}

	logOut, err = os.OpenFile(logDest, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
	if err != nil {
		panic(err)
	}

newLog:
	log = zap.New(
		zap.NewJSONEncoder(
			zap.RFC3339Formatter("ts"),
		),
		zap.Output(logOut),
	)
	log.SetLevel(*loglevel)
}

// CloseLog closes the log file
func CloseLog() {
	if logOut != nil {
		logOut.Close()
	}
}
