package main

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/mohae/autofac"
	//"github.com/mohae/autofac/util"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  autofac.ReadBufferSize,
	WriteBufferSize: autofac.WriteBufferSize,
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
		c.WriteMessage(websocket.TextMessage, []byte("invalid socket initiation request"))
		fmt.Fprintf(os.Stderr, "invalid initiation typ: %d\n", typ)
		return
	}
	fmt.Println("*** looking up client stuff ***")
	var cl *autofac.Client
	var message string
	var ok bool
	if len(b) == 0 {
		fmt.Println("*** new client ***")
		// get a new client and its ID
		cl = inventory.NewClient()
		fmt.Printf("new ID: %d\n", cl.ID)
		id := make([]byte, 4)
		binary.LittleEndian.PutUint32(id, cl.ID)
		err = c.WriteMessage(websocket.BinaryMessage, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing new client ID: %s\n", err)
		}
		goto listen
	}

	fmt.Println("*** existing client ***")
	message = fmt.Sprintf("welcome back %X", b)
	cl, ok = inventory.Client(binary.LittleEndian.Uint32(b))
	if !ok {
		cl = inventory.NewClient()
		// send the new client ID
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, cl.ID)
		err = c.WriteMessage(websocket.BinaryMessage, b)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing client ID for %X: %s\n", cl.ID, err)
			return
		}
		message = fmt.Sprintf("welcome back; could not fine %X in inventory, new id: %X\n", b, cl.ID)
	}
	// send the welcome message
	err = c.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing welcome message for %X: %s\n", cl.ID, err)
		return
	}

listen:
	for {
		typ, p, err := c.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading message: %s\n", err)
			return
		}
		// not doing anything with the type for now
		switch typ {
		case websocket.BinaryMessage:
			fmt.Printf("binary message: %x\n", p)
		case websocket.TextMessage:
			fmt.Printf("text message: %s\n", p)
		case websocket.PingMessage:
			fmt.Printf("ping message: %s\n", p)
		case websocket.PongMessage:
			fmt.Printf("pong message: %s\n", p)
		case websocket.CloseMessage:
			fmt.Printf("close message: %s\n", p)
		}
		err = c.WriteMessage(typ, p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing message: %s\n", err)
			return
		}
	}
}
