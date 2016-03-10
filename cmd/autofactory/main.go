package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mohae/autofact/client"
)

var srvr server

var connCfg client.ConnCfg

// the serialized default client.Cfg.  The data is originally loaded from the
// server's ClientCfg file, which is specified by clientCfgFile.
var clientCfg []byte
var clientCfgFile = flag.String("clientcfg", "autofact-client.json", "location of client configuration file")
var dbFile = flag.String("dbfile", "autofact.bdb", "location of the autofactory database file")
var influxDBName string
var influxUser string
var influxPassword string
var influxAddress string

// flags
func init() {
	flag.StringVar(&connCfg.ServerPort, "port", "8675", "port to use for websockets")
	flag.StringVar(&connCfg.ServerPort, "p", "8675", "port to use for websockets (short)")
	flag.StringVar(clientCfgFile, "c", "autofact-client.json", "location of client configuration file (short)")
	flag.StringVar(dbFile, "d", "autofact.bdb", "location of the autfactory database file (short)")
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

	// load the default client cfg
	var cCfg ClientCfg
	err := cCfg.Load(*clientCfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading the client configuration file: %s\n", err)
		return 1
	}
	clientCfg = cCfg.Serialize()
	srvr.ClientCfg = clientCfg
	// Ther server PingPeriod and PongWait should be the same as the clients
	srvr.PingPeriod = cCfg.PingPeriod
	srvr.PongWait = cCfg.PongWait

	// bdb is used as the extension for bolt db.
	err = srvr.DB.Open("autofact.bdb")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %s", err)
		return 1
	}
	defer srvr.DB.DB.Close()

	// connect to Influx
	err = srvr.connectToInfluxDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to %s: %s", influxDBName, err)
		return 1
	}
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
	fmt.Println("autofact: running")
	return 0
}
