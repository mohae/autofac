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
	var n *Node
	var msg string
	var ok, isNew bool
	// the bytes are ClientInf
	inf := client.GetRootAsInf(p, 0)
	if inf.ID() == 0 {
		isNew = true
		// get a new client and its ID
		n, err = srvr.NewNode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client")
			return
		}
		fmt.Printf("new ID: %d\n", n.Inf.ID())
		// sync the client's info with the new Node's
		// TODO: revisit when region/zone/dc is better implemented. i.e.
		// the client probably won't have this info on first connect, will it?
		bldr := flatbuffers.NewBuilder(0)
		h := bldr.CreateByteString(inf.Hostname())
		r := bldr.CreateByteString(inf.Region())
		z := bldr.CreateByteString(inf.Zone())
		d := bldr.CreateByteString(inf.DC())
		client.InfStart(bldr)
		client.InfAddID(bldr, n.Inf.ID())
		client.InfAddHostname(bldr, h)
		client.InfAddRegion(bldr, r)
		client.InfAddZone(bldr, z)
		client.InfAddDC(bldr, d)
		bldr.Finish(client.InfEnd(bldr))
		b := bldr.Bytes[bldr.Head():]
		err = c.WriteMessage(websocket.BinaryMessage, b)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing new client ID: %s\n", err)
		}
		n.Inf = client.GetRootAsInf(b, 0)
		// save the updated inf to the inventory
		srvr.Inventory.AddNodeInf(n.Inf.ID(), n.Inf)
		goto sendCfg
	}

	msg = fmt.Sprintf("welcome back %X\n", inf.ID())
	n, ok = srvr.Node(inf.ID())
	if !ok {
		isNew = true
		n, err = srvr.NewNode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client")
			return
		}
		bldr := flatbuffers.NewBuilder(0)
		h := bldr.CreateByteString(inf.Hostname())
		r := bldr.CreateByteString(inf.Region())
		z := bldr.CreateByteString(inf.Zone())
		d := bldr.CreateByteString(inf.DC())
		client.InfStart(bldr)
		client.InfAddID(bldr, n.Inf.ID())
		client.InfAddHostname(bldr, h)
		client.InfAddRegion(bldr, r)
		client.InfAddZone(bldr, z)
		client.InfAddDC(bldr, d)
		bldr.Finish(client.InfEnd(bldr))
		b := bldr.Bytes[bldr.Head():]
		n.Inf = client.GetRootAsInf(b, 0)
		err = c.WriteMessage(websocket.BinaryMessage, b)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing client ID for %X: %s\n", n.Inf.ID(), err)
			return
		}
		msg = fmt.Sprintf("welcome back; could not find %X in inventory, new id: %X\n", inf.ID(), n.Inf.ID())
	}
	// send the welcome message
	err = c.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing welcome message for %X: %s\n", n.Inf.ID(), err)
		return
	}

	// TODO: for existing client, send the cfg from the hydrated info
sendCfg:
	_ = isNew
	// the node needs the current connection
	n.WS = c
	// send the config
	n.WriteBinaryMessage(message.ClientCfg, srvr.ClientCfg)

	// set the ping hanlder
	n.WS.SetPingHandler(n.PingHandler)
	n.WS.SetPingHandler(n.PongHandler)
	// start a message handler for the client
	doneCh := make(chan struct{})
	go n.Listen(doneCh)

	// wait for the done signal
	<-doneCh
}
