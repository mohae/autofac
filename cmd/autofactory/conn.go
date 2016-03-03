package main

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/mohae/autofact"
	//"github.com/mohae/autofact/util"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  autofact.ReadBufferSize,
	WriteBufferSize: autofact.WriteBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func serveClient(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade error: %s\n", err)
	}
	defer c.Close()
	// first message is the clientID, if "" then get a new one
	typ, b, err := c.ReadMessage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading message: %s\n", err)
		return
	}
	// if messageType isn't BinaryMessage, reject
	if typ != websocket.BinaryMessage {
		c.WriteMessage(websocket.CloseMessage, []byte("invalid socket initiation request"))
		fmt.Fprintf(os.Stderr, "invalid initiation typ: %d\n", typ)
		return
	}
	fmt.Println("*** looking up client stuff ***")
	var cl *autofact.Client
	var message string
	var ok bool
	// decode the byte (should be len 4); if something else, reject
	if len(b) != 4 {
		c.WriteMessage(websocket.CloseMessage, []byte("invalid socket initiation request: malformed ID"))
		fmt.Fprintf(os.Stderr, "invalid socket initiation request: malformed ID\n")
		return
	}
	id := binary.LittleEndian.Uint32(b)
	if id == 0 {
		fmt.Println("*** new client ***")
		// get a new client and its ID
		cl = srvr.Inventory.NewClient()
		fmt.Printf("new ID: %d\n", cl.ID)
		id := make([]byte, 4)
		binary.LittleEndian.PutUint32(id, cl.ID)
		err = c.WriteMessage(websocket.BinaryMessage, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing new client ID: %s\n", err)
		}
		goto listen
	}

	message = fmt.Sprintf("welcome back %X\n", id)
	cl, ok = srvr.Inventory.Client(id)
	if !ok {
		cl = srvr.Inventory.NewClient()
		// send the new client ID
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, cl.ID)
		err = c.WriteMessage(websocket.BinaryMessage, b)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing client ID for %X: %s\n", cl.ID, err)
			return
		}
		message = fmt.Sprintf("welcome back; could not find %X in inventory, new id: %X\n", b, cl.ID)
	}
	// send the welcome message
	err = c.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing welcome message for %X: %s\n", cl.ID, err)
		return
	}

listen:
	// the client needs the current connection
	cl.WS = c
	// let the client struct know it is on the server side
	cl.SetIsServer(true)
	// set the ping hanlder
	cl.WS.SetPingHandler(cl.PingHandler)
	cl.WS.SetPingHandler(cl.PongHandler)
	// start a message handler for the client
	doneCh := make(chan struct{})
	go cl.Listen(doneCh)

	// wait for the done signal
	<-doneCh
}
