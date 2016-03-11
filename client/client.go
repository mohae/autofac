package client

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/message"
	"github.com/mohae/autofact/sysinfo"
)

// Client is anything that talks to the server.
type Client struct {
	// Inf holds the basic client information.
	*Inf
	// This is Inf as bytes: this is hopefully unnecessary, but I don't know at this point.
	InfBytes []byte
	// Conn holds the configuration for connecting to the server.
	ConnCfg
	// Cfg holds the client configuration (how the client behaves).
	*Cfg

	// Healthbeat buffers
	CPUData [][]byte
	MemData [][]byte

	muSend sync.Mutex
	WS     *websocket.Conn
	// Channel for outbound binary messages.  The message is assumed to be a
	// websocket.Binary type
	SendB       chan []byte
	SendStr     chan string
	mu          sync.Mutex
	isConnected bool
	ServerURL   url.URL
}

func New(id uint32, name string) *Client {
	bldr := flatbuffers.NewBuilder(0)
	n := bldr.CreateString(name)
	InfStart(bldr)
	InfAddID(bldr, id)
	InfAddHostname(bldr, n)
	bldr.Finish(InfEnd(bldr))
	return &Client{
		Inf:      GetRootAsInf(bldr.Bytes[bldr.Head():], 0),
		InfBytes: bldr.Bytes[bldr.Head():],
		// A really small buffer:
		// TODO: rethink this vis-a-vis what happens when recipient isn't there
		// or if it goes away during sending and possibly caching items to be sent.
		SendB:   make(chan []byte, 10),
		SendStr: make(chan string, 10),
	}
}

// Connect handles connecting to the server and returns the connection status.
// The client will attempt to connect until it has either succeeded or the
// connection retry period has been exceeded.  A retry is done every 5 seconds.
//
// If the client is already connected, nothing will be done.
func (c *Client) Connect() bool {
	// If already connected, return that fact.
	if c.IsConnected() {
		return true
	}
	start := time.Now()
	retryEnd := start.Add(c.ConnectPeriod)
	// connect to server; retry until the retry period has expired
	for {
		if time.Now().After(retryEnd) {
			fmt.Fprintf(os.Stderr, "timed out while trying to connect to the server: %s\n", c.ServerURL.String())
			return false
		}
		err := c.DialServer()
		if err == nil {
			break
		}
		time.Sleep(c.ConnectInterval)
		fmt.Printf("unable to connect to the server %s: retrying...\n", c.ServerURL.String())
	}
	// Send the ClientInf.  If the ID == 0 or it can't be found, the server will
	// respond with one.  Retry until the server responds, or until the
	// reconnectPeriod has expired.
	err := c.WS.WriteMessage(websocket.BinaryMessage, c.InfBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while sending ID: %s\n", err)
		c.WS.Close()
		return false
	}

	// read messages until we get an EOT
handshake:
	for {
		typ, p, err := c.WS.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while Reading ID response: %s\n", err)
			c.WS.Close()
			return false
		}

		switch typ {
		case websocket.BinaryMessage:
			// process according to message kind
			msg := message.GetRootAsMessage(p, 0)
			switch message.Kind(msg.Kind()) {
			case message.ClientInf:
				c.Inf = GetRootAsInf(msg.DataBytes(), 0)
			case message.ClientCfg:
				c.Cfg = GetRootAsCfg(msg.DataBytes(), 0)
			case message.EOT:
				break handshake
			default:
				fmt.Fprint(os.Stderr, "unknown message type received during handshake; quitting\n")
				return false
			}
		case websocket.TextMessage:
			fmt.Printf("%s\n", string(p))
		default:
			fmt.Printf("unexpected welcome response type %d: %v\n", typ, p)
			c.WS.Close()
			return false
		}
	}
	fmt.Printf("%X connected\n", c.Inf.ID())
	c.mu.Lock()
	c.isConnected = true
	c.mu.Unlock()
	return true
}

func (c *Client) DialServer() error {
	var err error
	c.WS, _, err = websocket.DefaultDialer.Dial(c.ServerURL.String(), nil)
	return err
}

func (c *Client) MessageWriter(doneCh chan struct{}) {
	pingPeriod := time.Duration(c.Cfg.PingPeriod())
	defer close(doneCh)
	for {
		select {
		case p, ok := <-c.SendB:
			// don't send if not connected
			if !c.IsConnected() {
				continue
			}
			if !ok {
				c.WS.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.WS.WriteMessage(websocket.BinaryMessage, p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing message: %s\n", err)
			}
		case <-time.After(pingPeriod):
			// only ping if we are connected
			if !c.IsConnected() {
				continue
			}
			err := c.WS.WriteMessage(websocket.PingMessage, []byte("ping"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "ping error: %s\n", err)
			}
		}
	}
}

func (c *Client) Reconnect() bool {
	c.mu.Lock()
	c.isConnected = false
	c.mu.Unlock()
	for i := 0; i < 4; i++ {
		b := c.Connect()
		if b {
			fmt.Println("reconnect true")
			return b
		}
	}
	return false
}

func (c *Client) Listen(doneCh chan struct{}) {
	// loop until there's a done signal
	defer close(doneCh)
	for {
		typ, p, err := c.WS.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading message: %s\n", err)
			if _, ok := err.(*websocket.CloseError); !ok {
				return
			}
			fmt.Println("reconnecting from read messages")
			connected := c.Reconnect()
			if connected {
				continue
			}
			fmt.Fprint(os.Stderr, "unable to reconnect to server")
			return
		}
		switch typ {
		case websocket.TextMessage:
			fmt.Printf("textmessage: %s\n", p)
			if bytes.Equal(p, autofact.AckMsg) {
				// if this is an acknowledgement message, do nothing
				continue
			}
			err := c.WS.WriteMessage(websocket.TextMessage, autofact.AckMsg)
			if err != nil {
				if _, ok := err.(*websocket.CloseError); !ok {
					return
				}
				fmt.Println("reconnect from writing message: textmessage")
				connected := c.Reconnect()
				if connected {
					continue
				}
				fmt.Fprint(os.Stderr, "unable to reconnect to server")
				return
			}
		case websocket.BinaryMessage:
			err = c.WS.WriteMessage(websocket.TextMessage, autofact.AckMsg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing binary message: %s\n", err)
				if _, ok := err.(*websocket.CloseError); !ok {
					return
				}
				fmt.Println("reconnect from writing message: binarymessage")
				connected := c.Reconnect()
				if connected {
					continue
				}
				fmt.Fprint(os.Stderr, "unable to reconnect to server")
				return
			}
			c.processBinaryMessage(p)
		case websocket.CloseMessage:
			fmt.Printf("closemessage: %x\n", p)
			fmt.Println("reconnect from writing message: closemessage")
			connected := c.Reconnect()
			if connected {
				continue
			}
			fmt.Fprint(os.Stderr, "unable to reconnect to server")
			return
		}
	}
}

// IsConnected returns if the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

func (c *Client) PingHandler(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Printf("ping: %s\n", msg)
	return c.WS.WriteMessage(websocket.PongMessage, []byte("ping"))
}

func (c *Client) PongHandler(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Printf("pong: %s\n", msg)
	return c.WS.WriteMessage(websocket.PingMessage, []byte("pong"))
}

// TODO: should CPUData be enclosed in a struct for locking purposes?
func (c *Client) EnqueueCPUData(data []byte) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CPUData = append(c.CPUData, data)
	return len(c.CPUData)
}

// EnqueueMemData adds the received data to the MemData buffer.  The current
// number of entries in the buffer is returned.
func (c *Client) EnqueueMemData(data []byte) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.MemData = append(c.MemData, data)
	return len(c.MemData)
}

// If the message send fails, whatever was cached will be lost.
// TODO: the current stats get copied for send; should the stat slice
// get reset now?  There is possible data loss this way; but there's
// possible data loss if the stat slice gets appended to between this
// copy op and the send completing, unless a lock is held the entire time,
// which could block the stats reading process leading to data loss due to
// missed reads.  I'm thinking COW or delete now; but punting becasue it
// really doesn't matter as this is just an experiment, right now.
// This also applies to CPUDatasFB
// TODO: should the messages to be sent be copied to a send cache so that
// there isn't data loss on a failed send?  Consecutive PushPeriods that
// failed to send may be problematic in that situation.
func (c *Client) FlushCPUData() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	data := make([][]byte, len(c.CPUData))
	copy(data, c.CPUData)
	c.CPUData = nil
	return data
}

// FlushMemData returns a copy of the client's MemData and nils the client's
// cache.
// The notes above apply here too.
func (c *Client) FlushMemData() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	data := make([][]byte, len(c.MemData))
	copy(data, c.MemData)
	c.MemData = nil
	return data
}

func (c *Client) Healthbeat() {
	// An interval of 0 means no healthbeat
	if c.Cfg.HealthbeatInterval() == 0 {
		return
	}
	cpuCh := make(chan []byte)
	memCh := make(chan []byte)
	go sysinfo.CPUDataTicker(time.Duration(c.Cfg.HealthbeatInterval()), cpuCh)
	go sysinfo.MemDataTicker(time.Duration(c.Cfg.HealthbeatInterval()), memCh)
	t := time.NewTicker(time.Duration(c.Cfg.HealthbeatPushPeriod()))
	defer t.Stop()
	for {
		select {
		case data, ok := <-cpuCh:
			if !ok {
				fmt.Println("cpu stats chan closed")
				goto done
			}
			c.EnqueueCPUData(data)
		case data, ok := <-memCh:
			if !ok {
				fmt.Println("cpu stats chan closed")
				goto done
			}
			c.EnqueueMemData(data)
		case <-t.C:
			if !c.IsConnected() {
				continue
			}
			c.SendData(message.CPUData, c.FlushCPUData())
			c.SendData(message.MemData, c.FlushMemData())
		}
	}
done:
	// Flush the buffer.
	c.SendData(message.CPUData, c.FlushCPUData())
	c.SendData(message.MemData, c.FlushMemData())
}

// SendData sends the received data to the server.  The caller checks to see
// if the client is connected to the server before calling.  If the connection
// is lost during processing, the cached stats will be lost.
// TODO:  should this be more resilient?
func (c *Client) SendData(kind message.Kind, data [][]byte) error {
	// for each data, send a message
	for _, v := range data {
		c.SendB <- message.Serialize(c.Inf.ID(), kind, v)
	}
	return nil
}

// SendMessage sends a single serialized message of type Kind.
func (c *Client) SendMessage(kind message.Kind, p []byte) {
	c.SendB <- message.Serialize(c.Inf.ID(), kind, p)
}

// binary messages are expected to be flatbuffer encoding of message.Message.
func (c *Client) processBinaryMessage(p []byte) error {
	// unmarshal the message
	msg := message.GetRootAsMessage(p, 0)
	// process according to kind
	k := message.Kind(msg.Kind())
	switch k {
	case message.ClientCfg:
		c.Cfg.Deserialize(msg.DataBytes())
	default:
		fmt.Println("unknown message kind")
		fmt.Println(string(p))
	}
	return nil
}

// Serialize serializes the Inf using flatbuffers and returns the []byte.
func (i *Inf) Serialize() []byte {
	bldr := flatbuffers.NewBuilder(0)
	h := bldr.CreateByteString(i.Hostname())
	r := bldr.CreateByteString(i.Region())
	z := bldr.CreateByteString(i.Zone())
	d := bldr.CreateByteString(i.DC())
	InfStart(bldr)
	InfAddID(bldr, i.ID())
	InfAddHostname(bldr, h)
	InfAddRegion(bldr, r)
	InfAddZone(bldr, z)
	InfAddDC(bldr, d)
	bldr.Finish(InfEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}
