package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/client"
	"github.com/mohae/autofact/message"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  autofact.ReadBufferSize,
	WriteBufferSize: autofact.WriteBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// serveClient takes a new client connection and either looks up the
// information for the client, or creates a new client and clientID ( in
// instances where the client has either never connected before or it's
// information cannot be found)
func serveClient(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade error: %s\n", err)
	}
	defer c.Close()
	// first message is the clientID, if "" then get a new one
	typ, p, err := c.ReadMessage()
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
	var cl *Client
	var msg string
	var ok, isNew bool
	// the bytes are ClientInf
	inf := client.GetRootAsClientInf(p, 0)
	if inf.ID() == 0 {
		isNew = true
		// get a new client and its ID
		cl, err = srvr.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client")
			return
		}
		cl.Hostname = string(inf.Hostname())
		fmt.Printf("new ID: %d\n", cl.ID)
		bldr := flatbuffers.NewBuilder(0)
		name := bldr.CreateString(cl.Hostname)
		client.ClientInfStart(bldr)
		client.ClientInfAddID(bldr, cl.ID)
		client.ClientInfAddHostname(bldr, name)
		bldr.Finish(client.ClientInfEnd(bldr))
		err = c.WriteMessage(websocket.BinaryMessage, bldr.Bytes[bldr.Head():])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing new client ID: %s\n", err)
		}
		goto sendCfg
	}

	msg = fmt.Sprintf("welcome back %X\n", inf.ID())
	cl, ok = srvr.Client(inf.ID())
	if !ok {
		isNew = true
		cl, err = srvr.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client")
			return
		}
		bldr := flatbuffers.NewBuilder(0)
		name := bldr.CreateString(cl.Hostname)
		client.ClientInfStart(bldr)
		client.ClientInfAddID(bldr, cl.ID)
		client.ClientInfAddHostname(bldr, name)
		bldr.Finish(client.ClientInfEnd(bldr))
		err = c.WriteMessage(websocket.BinaryMessage, bldr.Bytes[bldr.Head():])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing client ID for %X: %s\n", cl.ID, err)
			return
		}
		msg = fmt.Sprintf("welcome back; could not find %X in inventory, new id: %X\n", inf.ID(), cl.ID)
	}
	// send the welcome message
	err = c.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing welcome message for %X: %s\n", cl.ID, err)
		return
	}

	// TODO: for existing client, send the cfg from the hydrated info
sendCfg:
	_ = isNew
	// the client needs the current connection
	cl.WS = c

	// send the config
	cl.WriteBinaryMessage(message.ClientCfg, srvr.ClientCfg)

	// set the ping hanlder
	cl.WS.SetPingHandler(cl.PingHandler)
	cl.WS.SetPingHandler(cl.PongHandler)
	// start a message handler for the client
	doneCh := make(chan struct{})
	go cl.Listen(doneCh)

	// wait for the done signal
	<-doneCh
}
