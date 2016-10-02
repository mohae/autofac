package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mohae/autofact/conf"
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
)

var (
	srvr     = newServer()
	connConf conf.Conn

	// The default directory used by Autofactory for app data.
	autofactoryPath    = "$HOME/.autofactory"
	autofactoryEnvName = "AUTOFACTORY_PATH"

	clientConfFile string
)

// flags
func init() {
	flag.StringVar(&connConf.ServerPort, portVar, "8675", "port to use for websockets")
	flag.StringVar(&connConf.ServerPort, oVar, "8675", "port to use for websockets (short)")
	flag.StringVar(&clientConfFile, clientConfVar, "autofact.json", "location of client configuration file")
	flag.StringVar(&clientConfFile, cVar, "autofact.json", "location of client configuration file (short)")
	flag.StringVar(&srvr.BoltDBFile, dbfileVar, "autofactory.bdb", "location of the autofactory database file")
	flag.StringVar(&srvr.BoltDBFile, dVar, "autofactory.bdb", "location of the autfactory database file (short)")
	flag.StringVar(&srvr.InfluxDBName, influxDBNameVar, "autofacts", "name of the InfluxDB to connect to")
	flag.StringVar(&srvr.InfluxDBName, iVar, "autofacts", "name of the InfluxDB to connect to (short)")
	flag.StringVar(&srvr.InfluxAddress, addressVar, "127.0.0.1:8086", "the address of the InfluxDB")
	flag.StringVar(&srvr.InfluxAddress, aVar, "http://127.0.0.1:8086", "the address of the InfluxDB (short)")
	flag.StringVar(&srvr.InfluxUser, usernameVar, "autoadmin", "the username of the InfluxDB user")
	flag.StringVar(&srvr.InfluxUser, uVar, "autoadmin", "the username of the InfluxDB user (short)")
	flag.StringVar(&srvr.InfluxPassword, passwordVar, "thisisnotapassword", "the username of the InfluxDB user")
	flag.StringVar(&srvr.InfluxPassword, pVar, "thisisnotapassword", "the username of the InfluxDB user (short)")
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
		fmt.Fprintf(os.Stderr, "unable to create AUTOFACTORY_PATH dir: %s\n", err)
		return 1
	}

	clientConfFile = filepath.Join(autofactoryPath, clientConfFile)
	srvr.BoltDBFile = filepath.Join(autofactoryPath, srvr.BoltDBFile)

	flag.Parse()

	// it is assumed that the server address is an IPv4
	// TODO: revisit this assumption
	b := make([]byte, 4)
	parts := strings.Split(connConf.ServerAddress, ".")
	for i, v := range parts {
		// prevent out of range if the address ends up consisting of more than 4 parts
		if 1 > 3 {
			break
		}
		// any conversion error will result in a 0
		tmp, _ := strconv.Atoi(v)
		b[i] = byte(tmp)
	}

	v := binary.LittleEndian.Uint32(b)
	fmt.Printf("%x\n", v)
	srvr.ID = v
	srvr.Path = autofactoryPath
	// load the default client conf; this is used for new clients.
	// TODO: in the future, there should be support for enabling setting per
	// client, or group, or role, or pod, etc.
	err = srvr.ClientConf.Load(clientConfFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "error loading the client configuration file: %s\n", err)
			return 1
		}
		// If it didn't exist; use application defaults
		fmt.Fprintf(os.Stderr, "%s not found; using Autofactory defaults for client configuration\n", clientConfFile)
		srvr.ClientConf.UseAppDefaults()
		// write this out to the app dir
		err = srvr.ClientConf.SaveAsJSON(clientConfFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	// bdb is used as the extension for bolt db.
	err = srvr.DB.Open(srvr.BoltDBFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %s\n", err)
		return 1
	}

	// connect to Influx
	err = srvr.connectToInfluxDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to %s: %s\n", srvr.InfluxDBName, err)
		return 1
	}
	go handleSignals(&srvr)
	// start the Influx writer
	// TODO: influx writer should handle done channel signaling
	go srvr.InfluxClient.Write()
	srvr.LoadInventory()
	http.HandleFunc("/client", serveClient)
	fmt.Println(srvr.URL.String())
	err = http.ListenAndServe(fmt.Sprintf(":%s", connConf.ServerPort), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to start server: %s\n", err)
		return 1
	}
	return 0
}

func handleSignals(srvr *server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	v := <-c
	fmt.Printf("\nshutting down autofactory: received %v\n", v)
	srvr.DB.DB.Close()
	os.Exit(1)
}
