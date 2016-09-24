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

	"github.com/mohae/autofact/cfg"
)

var (
	srvr    server
	connCfg cfg.Conn

	// The default directory used by Autofactory for app data.
	autofactoryPath    = "$HOME/.autofactory"
	autofactoryEnvName = "AUTOFACTORY_PATH"

	// the serialized default client.Cfg.  The data is originally loaded from the
	// server's ClientCfg file, which is specified by clientCfgFile.
	clientConf     []byte
	clientConfFile string
	bDBFile        string
	influxDBName   string
	influxUser     string
	influxPassword string
	influxAddress  string
)

// flags
func init() {
	flag.StringVar(&connCfg.ServerPort, "port", "8675", "port to use for websockets")
	flag.StringVar(&connCfg.ServerPort, "p", "8675", "port to use for websockets (short)")
	flag.StringVar(&clientConfFile, "clientcfg", "autofact.json", "location of client configuration file")
	flag.StringVar(&clientConfFile, "c", "autofact.json", "location of client configuration file (short)")
	flag.StringVar(&bDBFile, "dbfile", "autofactory.bdb", "location of the autofactory database file")
	flag.StringVar(&bDBFile, "d", "autofactory.bdb", "location of the autfactory database file (short)")
	flag.StringVar(&influxDBName, "dbname", "autofacts", "name of the InfluxDB to connect to")
	flag.StringVar(&influxDBName, "n", "autofacts", "name of the InfluxDB to connect to (short)")
	flag.StringVar(&influxAddress, "address", "127.0.0.1:8086", "the address of the InfluxDB")
	flag.StringVar(&influxAddress, "a", "http://127.0.0.1:8086", "the address of the InfluxDB (short)")
	flag.StringVar(&influxUser, "username", "autoadmin", "the username of the InfluxDB user")
	flag.StringVar(&influxUser, "u", "autoadmin", "the username of the InfluxDB user (short)")
	flag.StringVar(&influxPassword, "password", "thisisnotapassword", "the username of the InfluxDB user")
	flag.StringVar(&influxPassword, "P", "thisisnotapassword", "the username of the InfluxDB user (short)")
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
	bDBFile = filepath.Join(autofactoryPath, bDBFile)

	flag.Parse()

	// it is assumed that the server address is an IPv4
	// TODO: revisit this assumption
	b := make([]byte, 4)
	parts := strings.Split(connCfg.ServerAddress, ".")
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
	srvr = newServer(v)
	srvr.Path = autofactoryPath
	// load the default client conf; this is used for new clients.
	// TODO: in the future, there should be support for enabling setting per
	// client, or group, or role, or pod, etc.
	var cConf ClientCfg
	err = cConf.Load(clientConfFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "error loading the client configuration file: %s\n", err)
			return 1
		}
		// If it didn't exist; use application defaults
		fmt.Fprintf(os.Stderr, "%s not found; using Autofactory defaults for client configuration\n", clientConfFile)
		cConf.UseAppDefaults()
		// write this out to the app dir
		err = cConf.SaveAsJSON(clientConfFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	clientConf = cConf.Serialize()
	srvr.ClientConf = clientConf

	// bdb is used as the extension for bolt db.
	err = srvr.DB.Open(bDBFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %s\n", err)
		return 1
	}

	// connect to Influx
	err = srvr.connectToInfluxDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to %s: %s\n", influxDBName, err)
		return 1
	}
	go handleSignals(&srvr)
	// start the Influx writer
	// TODO: influx writer should handle done channel signaling
	go srvr.InfluxClient.Write()
	srvr.LoadInventory()
	http.HandleFunc("/client", serveClient)
	fmt.Println(srvr.URL.String())
	err = http.ListenAndServe(fmt.Sprintf(":%s", connCfg.ServerPort), nil)
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
