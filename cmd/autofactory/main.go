package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/mohae/autofact/conf"
	czap "github.com/mohae/zap"
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
	logOut   string
	logFile  *os.File

	// Data; if data destination == file
	data     czap.Logger // use mohae's fork to support level description override
	dataOut  string
	dataFile *os.File
	dataDest string

	// if data destination == influxdb
	serverID       string
	clientConfFile string
	influxUser     string
	influxPassword string

	// The default directory used by Autofactory for app data.
	autofactoryPath    = "$HOME/.autofactory"
	autofactoryEnvName = "AUTOFACTORY_PATH"
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
	flag.StringVar(&logOut, "logout", "stderr", "log output; if empty stderr will be used")
	flag.StringVar(&logOut, "l", "stderr", "log output; if empty stderr will be used")
	flag.StringVar(&dataOut, "dataout", "stdout", "data output location for when the data destination is file, if empty stdout will be used")
	flag.StringVar(&dataDest, "datadestination", "file", "the destination for collected data: file or influxdb")
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
	defer CloseOut()

	srvr.ID = []byte(serverID)
	srvr.NewSnowflakeGenerator()
	srvr.AutoPath = autofactoryPath
	// load the default client conf; this is used for new clients.
	// TODO: in the future, there should be support for enabling setting per
	// client, or group, or role, or pod, etc.  Or should this be a custom list
	// of attributes that can be created?
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

	// Check data destination and handle accordingly
	switch dataDest {
	case "file":
		err = SetDataOut()
		if err != nil { // don't do anything with error, func already handled logging.
			return 1
		}

	case "influxdb":
		err = ConnectToInflux()
		if err != nil { // don't do anything with error, func already handled logging.
			return 1
		}

	default:
		fmt.Fprintf(os.Stderr, "fatal error: unsupported data destination %s", dataDest)
		return 1
	}

	go handleSignals(&srvr)
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
	CloseOut()
	os.Exit(1)
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

// CloseOut closes the output files.  This should be called before any os.Exit.
// Log.Fatal or Log.Panic, or anything else that doesn't allow defers to run
// or allow one to log the error and close the output files.
func CloseOut() {
	if logFile != nil {
		logFile.Close()
	}
	if dataFile != nil {
		dataFile.Close()
	}
	if srvr.Bolt.DB != nil {
		srvr.Bolt.Close()
	}
}

func SetDataOut() error {
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
		log.Error(
			err.Error(),
			zap.String("op", "open datafile"),
			zap.String("filename", dataOut),
		)
		return err
	}
newData:
	data = czap.New(
		czap.NewJSONEncoder(
			czap.RFC3339Formatter("ts"),
		),
		czap.Output(dataFile),
	)
	data.SetLevel(czap.WarnLevel)
	return nil
}

func ConnectToInflux() error {
	// connect to Influx
	err := srvr.connectToInfluxDB(influxUser, influxPassword)
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "connect to influxdb"),
			zap.String("db", srvr.InfluxDBName),
		)
		return err
	}
	// start the Influx writer
	// TODO: influx writer should handle done channel signaling
	go srvr.InfluxClient.Write()

	return nil
}
