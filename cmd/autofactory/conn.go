package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/cfg"
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
	var ok bool
	// the bytes are cfg.SysInf
	inf := cfg.GetRootAsSysInf(p, 0)
	if inf.ID() == 0 {
		// get a new client and its ID
		n, err = srvr.NewNode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client")
			return
		}
		goto sendInf
	}

	n, ok = srvr.Node(inf.ID())
	if !ok {
		n, err = srvr.NewNode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client")
			return
		}
	}

sendInf:
	// update the node with the current inf
	bldr := flatbuffers.NewBuilder(0)
	h := bldr.CreateByteString(inf.Hostname())
	rr := bldr.CreateByteString(inf.Region())
	z := bldr.CreateByteString(inf.Zone())
	d := bldr.CreateByteString(inf.DataCenter())
	cfg.SysInfStart(bldr)
	cfg.SysInfAddID(bldr, n.SysInf.ID())
	cfg.SysInfAddHostname(bldr, h)
	cfg.SysInfAddRegion(bldr, rr)
	cfg.SysInfAddZone(bldr, z)
	cfg.SysInfAddDataCenter(bldr, d)
	bldr.Finish(cfg.SysInfEnd(bldr))
	b := bldr.Bytes[bldr.Head():]
	n.SysInf = cfg.GetRootAsSysInf(b, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing client ID for %X: %s\n", n.SysInf.ID(), err)
		return
	}

	fmt.Printf("%X connected\n", n.SysInf.ID())

	// save the node inf to the inventory
	srvr.Inventory.SaveNodeInf(n.SysInf, b)
	// the node needs the current connection
	n.WS = c
	// send the inf
	n.WriteBinaryMessage(message.SysInf, b)
	// send the default config
	n.WriteBinaryMessage(message.ClientCfg, srvr.ClientCfg)
	// send EOM
	n.WriteBinaryMessage(message.EOT, nil)
	// set the ping hanlder
	n.WS.SetPingHandler(n.PingHandler)
	n.WS.SetPingHandler(n.PongHandler)
	// start a message handler for the client
	doneCh := make(chan struct{})
	go n.Listen(doneCh)

	// wait for the done signal
	<-doneCh
}
