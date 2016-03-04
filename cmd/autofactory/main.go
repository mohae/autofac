package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net/http"
	"os"

	_ "github.com/gorilla/websocket"
	_ "github.com/mohae/autofact/util"
)

// flags
var (
	addr = flag.String("addr", "127.0.0.1:8675", "")
)

var srvr server

func main() {
	os.Exit(realMain())
}

func realMain() int {
	flag.Parse()

	// todo: parse the addr to make this
	b := make([]byte, 4)
	b[0] = 127
	b[1] = 0
	b[2] = 0
	b[3] = 1
	v := binary.LittleEndian.Uint32(b)
	fmt.Printf("%x\n", v)
	srvr = newServer(v)

	// bdb is used as the extension for bolt db.
	err := srvr.DB.Open("autofact.bdb")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %s", err)
		return 1
	}
	defer srvr.DB.DB.Close()
	srvr.LoadInventory()
	http.HandleFunc("/client", serveClient)
	err = http.ListenAndServe(fmt.Sprintf("%s", *addr), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to start server: %s\n", err)
		return 1
	}
	fmt.Println("autofact: running")
	return 0
}
