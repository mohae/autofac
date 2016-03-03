package autofact

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/mohae/autofact/message"
	"github.com/mohae/autofact/sysinfo"
)

// Client is anything that talks to the server.
type Client struct {
	ID         uint32   `json:"id"`
	ServerID   uint32   `json:"server_id"`
	Datacenter string   `json:"datacenter"`
	Groups     []string `json:"groups"`
	Roles      []string `json:"roles"`
	ServerURL  url.URL  `json:"server_url"`
	// The interval to check system info: 0 means don't check.
	HealthBeatPeriod time.Duration `json:"health_beat_period"`
	// PushPeriod is the interval to push accumulated data to the server.
	// If the server connection is down; nothing will be pushed and the
	// data will continue to accumulate on the client side.
	PushPeriod time.Duration `json:"PushPeriod"`
	// Current cache for accumulated CPU Stats using flatbuffers
	CPUstats        [][]byte      `json:"cpu_stats_fb"`
	ReconnectPeriod time.Duration `json:"reconnect_period"`

	WS *websocket.Conn `json:"-"`
	// Channel for outbound binary messages.  The message is assumed to be a
	// websocket.Binary type
	SendB       chan []byte   `json:"-"`
	SendStr     chan string   `json:"-"`
	PingPeriod  time.Duration `json:"-"`
	PongWait    time.Duration `json:"-"`
	WriteWait   time.Duration `json:"-"`
	mu          sync.Mutex
	isServer    bool
	isConnected bool
}

func NewClient(id uint32) *Client {
	return &Client{
		ID:         id,
		PingPeriod: PingPeriod,
		PongWait:   PongWait,
		WriteWait:  WriteWait,
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
	// if this is the server, don't try to reconnect-that's the client's responsibility
	// TODO: is this the place to note client dc time for notification purposes?
	if c.isServer {
		return false
	}
	start := time.Now()
	retryEnd := start.Add(c.ReconnectPeriod)
	// connect to server; retry until the retry period has expired
	for {
		if time.Now().After(retryEnd) {
			fmt.Fprintln(os.Stderr, "timed out while trying to connect to the server")
			return false
		}
		err := c.DialServer()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
		fmt.Println("unable to connect to the server: retrying...")
	}
	// Send the client's ID; if it's empty or can't be found, the server will
	// respond with one.  Retry until the server responds, or until the
	// reconnectPeriod has expired.
	var err error
	var typ int
	var p []byte
	b := make([]byte, 4)
	if c.ID > 0 {
		binary.LittleEndian.PutUint32(b, c.ID)
	}
	err = c.WS.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while sending ID: %s\n", err)
		c.WS.Close()
		return false
	}

	typ, p, err = c.WS.ReadMessage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while Reading ID response: %s\n", err)
		c.WS.Close()
		return false
	}

	fmt.Printf("hello response: %d: %v\n", typ, p)
	switch typ {
	case websocket.BinaryMessage:
		// a binary response is a clientID
		c.ID = binary.LittleEndian.Uint32(p[:4])
		fmt.Printf("new ID: %d\n", c.ID)
	case websocket.TextMessage:
		fmt.Printf("%s\n", string(p))
	default:
		fmt.Printf("unexpected welcome response type %d: %v\n", typ, p)
		c.WS.Close()
		return false
	}
	c.mu.Lock()
	c.isConnected = true
	c.mu.Unlock()
	fmt.Println("return is connected == true")
	return true
}

func (c *Client) DialServer() error {
	var err error
	c.WS, _, err = websocket.DefaultDialer.Dial(c.ServerURL.String(), nil)
	return err
}

func (c *Client) MessageWriter(doneCh chan struct{}) {
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
		case <-time.After(c.PingPeriod):
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
	for i := 0; i < 10; i++ {
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
	ackMsg := []byte("message received")
	defer close(doneCh)

	for {
		fmt.Println("read message")
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
			if bytes.Equal(p, ackMsg) {
				// if this is an acknowledgement message, do nothing
				continue
			}
			err := c.WS.WriteMessage(websocket.TextMessage, []byte("message received"))
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
			if err != nil {
				fmt.Fprintf(os.Stderr, "error unmarshaling JSON into a message: %s\n", err)
				return
			}
			err = c.WS.WriteMessage(websocket.TextMessage, []byte("message received"))
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

func (c *Client) SetIsServer(b bool) {
	c.isServer = b
}

func (c *Client) IsServer() bool {
	return c.isServer
}

// IsConnected returns if the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

func (c *Client) PingHandler(msg string) error {
	fmt.Printf("ping: %s\n", msg)
	return c.WS.WriteMessage(websocket.PongMessage, []byte("ping"))
}

func (c *Client) PongHandler(msg string) error {
	fmt.Printf("pong: %s\n", msg)
	return c.WS.WriteMessage(websocket.PingMessage, []byte("pong"))
}

// TODO: should cpustats be enclosed in a struct for locking purposes?
func (c *Client) AddCPUStats(stats []byte) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CPUstats = append(c.CPUstats, stats)
	return len(c.CPUstats)
}

// If the message send fails, whatever was cached will be lost.
// TODO: the current stats get copied for send; should the stat slice
// get reset now?  There is possible data loss this way; but there's
// possible data loss if the stat slice gets appended to between this
// copy op and the send completing, unless a lock is held the entire time,
// which could block the stats reading process leading to data loss due to
// missed reads.  I'm thinking COW or delete now; but punting becasue it
// really doesn't matter as this is just an experiment, right now.
// This also applies to CPUStatsFB
// TODO: should the messages to be sent be copied to a send cache so that
// there isn't data loss on a failed send?  Consecutive PushPeriods that
// failed to send may be problematic in that situation.
func (c *Client) CPUStats() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats := make([][]byte, len(c.CPUstats))
	copy(stats, c.CPUstats)
	c.CPUstats = nil
	return stats
}

func (c *Client) HealthBeat() {
	fmt.Println("started HealthBeat")
	if c.HealthBeatPeriod == 0 {
		return
	}
	cpuCh := make(chan []byte)
	go sysinfo.CPUStatsTicker(c.HealthBeatPeriod, cpuCh)
	t := time.NewTicker(c.PushPeriod)
	defer t.Stop()
	for {
		select {
		case stats, ok := <-cpuCh:
			if !ok {
				fmt.Println("cpu stats chan closed")
				goto done
			}
			c.AddCPUStats(stats)
		case <-t.C:
			if !c.IsConnected() {
				continue
			}
			fmt.Println("time to send the cpu stats")
			c.SendCPUStats()
		}
	}
done:
	// Flush the buffer.
	c.SendCPUStats()
}

// SendCPUStatsFB sends the cached cpu stats to the server.  The caller
// checks to see if the client is connected to the server before calling.
// If the connection is lost during processing, the cached stats will be lost.
// TODO:  should this be more resilient?
func (c *Client) SendCPUStats() error {
	// Get a copy of the stats
	stats := c.CPUStats()
	fmt.Printf("cpustats: %d messages to send\n", len(stats))
	bldr := flatbuffers.NewBuilder(0)
	// for each stat, send a message
	for i, stat := range stats {
		id := bldr.CreateByteVector(message.NewMessageID(c.ID))
		data := bldr.CreateByteVector(stat)
		message.MessageStart(bldr)
		message.MessageAddID(bldr, id)
		message.MessageAddType(bldr, websocket.BinaryMessage)
		message.MessageAddKind(bldr, int16(message.CPUStat))
		message.MessageAddData(bldr, data)
		bldr.Finish(message.MessageEnd(bldr))
		c.SendB <- bldr.Bytes[bldr.Head():]
		fmt.Fprintf(os.Stdout, "CPUStats: messages %d sent\n", i+1)
		bldr.Reset()
	}
	// TODO: only reset the stats if the send was received by the server
	c.ResetCPUStats()
	return nil
}

// TODO: is this obsolete now that copying the stats to the sending process
// does this?
func (c *Client) ResetCPUStats() {
	c.mu.Lock()
	c.CPUstats = nil
	c.mu.Unlock()
}

// binary messages are expected to be flatbuffer encoding of message.Message.
func (c *Client) processBinaryMessage(p []byte) error {
	// unmarshal the message
	msg := message.GetRootAsMessage(p, 0)
	// process according to kind
	k := message.Kind(msg.Kind())
	switch k {
	case message.CPUStat:
		s := sysinfo.UnmarshalCPUStatsToString(msg.DataBytes())
		fmt.Println(s)
	default:
		fmt.Println("unknown message kind")
		fmt.Println(string(p))
	}
	return nil
}
