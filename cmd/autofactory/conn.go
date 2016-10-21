package main

import (
	"net/http"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/conf"
	"github.com/mohae/autofact/message"
	"github.com/mohae/autofact/util"
	"github.com/uber-go/zap"
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
		log.Error(
			err.Error(),
			zap.String("op", "upgrade client connection"),
		)
		return
	}
	defer conn.Close()
	// first message is the clientID, if "" then get a new one
	typ, p, err := conn.ReadMessage()
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "read new connection message"),
		)
		return
	}
	// if messageType isn't TextMessage, reject.  It should either be empty or
	// have a client ID.
	if typ != websocket.TextMessage {
		conn.WriteMessage(websocket.CloseMessage, []byte("invalid socket initiation request"))
		log.Error(
			"invalid connection initation type",
			zap.String("type", util.WSString(typ)),
		)
		return
	}
	var c *Client
	var ok bool
	if len(p) == 0 {
		// get a new client and its ID
		c, err = srvr.NewClient()
		if err != nil {
			log.Error(
				err.Error(),
				zap.String("op", "create client"),
			)
			return
		}
		goto sendInf
	}

	c, ok = srvr.Client(p)
	if !ok {
		c, err = srvr.NewClient()
		if err != nil {
			log.Error(
				err.Error(),
				zap.String("op", "create client"),
			)
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
	conf.ClientAddHealthbeatPeriod(bldr, c.Conf.HealthbeatPeriod())
	conf.ClientAddMemInfoPeriod(bldr, c.Conf.MemInfoPeriod())
	conf.ClientAddNetUsagePeriod(bldr, c.Conf.NetUsagePeriod())
	conf.ClientAddCPUUtilizationPeriod(bldr, c.Conf.CPUUtilizationPeriod())
	bldr.Finish(conf.ClientEnd(bldr))
	b := bldr.Bytes[bldr.Head():]
	c.Conf = conf.GetRootAsClient(b, 0)
	if err != nil {
		log.Error(
			err.Error(),
			zap.String("op", "send message"),
			zap.String("message type", "client configuration"),
		)
		return
	}

	log.Info(
		"client connected",
		zap.String("id", string(c.Conf.IDBytes())),
	)

	// Add the client inf to the inventory
	srvr.Inventory.AddClient(c.Conf)
	// the client needs the current connection
	c.WS = conn
	// send the inf
	srvr.WriteBinaryMessage(string(c.Conf.IDBytes()), c.WS, message.ClientConf, b)
	// send EOM
	srvr.WriteBinaryMessage(string(c.Conf.IDBytes()), c.WS, message.EOT, nil)
	// start a message handler for the client
	doneCh := make(chan struct{})
	go c.Listen(doneCh)
	go c.Healthbeat(doneCh)
	// wait for the done signal
	<-doneCh
}
