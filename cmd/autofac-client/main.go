package main

import (
    "flag"
    "fmt"
    "net/url"
    "os"

    "github.com/gorilla/websocket"
    _ "github.com/mohae/autofac"
)

// flags
var (
	addr = flag.String("addr", "127.0.0.1:8675", "")
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	flag.Parse()

    // connect to the Server
    u := url.URL{Scheme: "ws", Host: *addr, Path: "/client"}
    fmt.Printf("connecting to %s\n", u)
    c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error connecting to %s: %s\n", u.String(), err)
        return 1
    }
    // initiate request
    err = c.WriteMessage(websocket.BinaryMessage, nil)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error writing message: %s", err)
        return 1
    }

    for {
        fmt.Println("*** reading message ***")
        typ, v, err := c.ReadMessage()
        if err != nil {
            fmt.Fprintf(os.Stderr, "error reading message: %s", err)
            return 1
        }
        switch typ {
        case websocket.TextMessage:
            fmt.Println(string(v))
        case websocket.BinaryMessage:
            fmt.Printf("%X\n", v)
        case websocket.PingMessage:
            fmt.Println("ping message")
        case websocket.PongMessage:
            fmt.Println("pong message")
        case websocket.CloseMessage:
            fmt.Println("close message")
        }
    }
//    cl := autofac.NewClient(c)
//    defer cl.Close()
    return 0
}
