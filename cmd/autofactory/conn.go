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
	// if messageType isn't TextMessage, reject.  It should either be empty or
	// have a client ID.
	if typ != websocket.TextMessage {
		conn.WriteMessage(websocket.CloseMessage, []byte("invalid socket initiation request"))
		fmt.Fprintf(os.Stderr, "invalid initiation typ: %d\n", typ)
		return
	}
	var c *Client
	var ok bool
	if len(p) == 0 {
		fmt.Println("new clientid")
		// get a new client and its ID
		c, err = srvr.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client: %s\n", err)
			return
		}
		goto sendInf
	}

	c, ok = srvr.Client(p)
	if !ok {
		c, err = srvr.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create new client: %s\n", err)
			return
		}
	}

sendInf:
	// update the node with the current inf
	bldr := flatbuffers.NewBuilder(0)
	h := bldr.CreateByteString(c.Conf.Hostname())
	rr := bldr.CreateByteString(c.Conf.Region())
	z := bldr.CreateByteString(c.Conf.Zone())
	d := bldr.CreateByteString(c.Conf.DataCenter())
	id := bldr.CreateByteVector(c.Conf.IDBytes())
	conf.ClientStart(bldr)
	conf.ClientAddID(bldr, id)
	conf.ClientAddHostname(bldr, h)
	conf.ClientAddRegion(bldr, rr)
	conf.ClientAddZone(bldr, z)
	conf.ClientAddDataCenter(bldr, d)
	bldr.Finish(conf.ClientEnd(bldr))
	b := bldr.Bytes[bldr.Head():]
	c.Conf = conf.GetRootAsClient(b, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing client ID for %s: %s\n", string(c.Conf.IDBytes()), err)
		return
	}

	fmt.Printf("%s connected\n", string(c.Conf.IDBytes()))

	// save the client inf to the inventory
	srvr.Inventory.SaveClient(c.Conf, b)
	// the client needs the current connection
	c.WS = conn
	// send the inf
	c.WriteBinaryMessage(message.SysInf, b)
	// send the client info
	c.WriteBinaryMessage(message.ClientConf, b)
	// send EOM
	c.WriteBinaryMessage(message.EOT, nil)
	// start a message handler for the client
	doneCh := make(chan struct{})
	go c.Listen(doneCh)

	// wait for the done signal
	<-doneCh
}
