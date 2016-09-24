package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/conf"
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade error: %s\n", err)
	}
	defer conn.Close()
	// first message is the clientID, if "" then get a new one
	typ, p, err := conn.ReadMessage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading message: %s\n", err)
		return
	}
	// if messageType isn't BinaryMessage, reject
	if typ != websocket.BinaryMessage {
		conn.WriteMessage(websocket.CloseMessage, []byte("invalid socket initiation request"))
		fmt.Fprintf(os.Stderr, "invalid initiation typ: %d\n", typ)
		return
	}
	var c *Client
	var ok bool
	// the bytes are conf.Node
	n := conf.GetRootAsNode(p, 0)
	if n.ID() == 0 {
		// get a new client and its ID
		c, err = srvr.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client")
			return
		}
		goto sendInf
	}

	c, ok = srvr.Client(n.ID())
	if !ok {
		c, err = srvr.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client")
			return
		}
	}

sendInf:
	// update the node with the current inf
	bldr := flatbuffers.NewBuilder(0)
	h := bldr.CreateByteString(n.Hostname())
	rr := bldr.CreateByteString(n.Region())
	z := bldr.CreateByteString(n.Zone())
	d := bldr.CreateByteString(n.DataCenter())
	conf.NodeStart(bldr)
	conf.NodeAddID(bldr, c.Node.ID())
	conf.NodeAddHostname(bldr, h)
	conf.NodeAddRegion(bldr, rr)
	conf.NodeAddZone(bldr, z)
	conf.NodeAddDataCenter(bldr, d)
	bldr.Finish(conf.NodeEnd(bldr))
	b := bldr.Bytes[bldr.Head():]
	c.Node = conf.GetRootAsNode(b, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing client ID for %X: %s\n", c.Node.ID(), err)
		return
	}

	fmt.Printf("%X connected\n", c.Node.ID())

	// save the client inf to the inventory
	srvr.Inventory.SaveNode(c.Node, b)
	// the client needs the current connection
	c.WS = conn
	// send the inf
	c.WriteBinaryMessage(message.SysInf, b)
	// send the default config
	c.WriteBinaryMessage(message.ClientConf, srvr.ClientConf)
	// send EOM
	c.WriteBinaryMessage(message.EOT, nil)
	// start a message handler for the client
	doneCh := make(chan struct{})
	go c.Listen(doneCh)

	// wait for the done signal
	<-doneCh
}
