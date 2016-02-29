package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net/http"
	"os"

	_ "github.com/gorilla/websocket"
	_ "github.com/mohae/autofac/util"
)

// flags
var (
	addr = flag.String("addr", "127.0.0.1:8675", "")
)

// global: currently
var inventory = NewInventory()

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
	fmt.Println(v)
	s := NewServer(v)
	_ = s
	http.HandleFunc("/client", serveClient)
	err := http.ListenAndServe(fmt.Sprintf("%s", *addr), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to start server: %s\n", err)
		return 1
	}
	fmt.Println("autofac: running")
	return 0
}
