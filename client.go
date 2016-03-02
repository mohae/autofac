package autofact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

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
	// TODO: the two versions of CPU stats are probably temporary.
	// Current cache for accumulated CPU Stats.
	CPUstats []sysinfo.CPUStat `json:"cpu_stats"`
	// Current cache for accumulated CPU Stats using flatbuffers
	CPUstatsFB [][]byte        `json:"cpu_stats_fb"`
	WS         *websocket.Conn `json:"-"`
	// channel for outbound messages
	Send       chan message.Message `json:"-"`
	PingPeriod time.Duration        `json:"-"`
	PongWait   time.Duration        `json:"-"`
	WriteWait  time.Duration        `json:"-"`
	mu         sync.Mutex
	isServer   bool
}

func NewClient(id uint32) *Client {
	return &Client{
		ID:         id,
		PingPeriod: PingPeriod,
		PongWait:   PongWait,
		WriteWait:  WriteWait,
		Send:       make(chan message.Message, 10),
	}
}

func (c *Client) DialServer() error {
	var err error
	c.WS, _, err = websocket.DefaultDialer.Dial(c.ServerURL.String(), nil)
	return err
}

func (c *Client) Close() error {
	return c.WS.Close()
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
				fmt.Fprintf(os.Stderr, "error writing text message: %s\n", err)
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
				return
			}
			c.processBinaryMessage(p)
		case websocket.CloseMessage:
			fmt.Printf("closemessage: %x\n", p)
			return
		}
	}
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
func (c *Client) AddCPUStats(stats []sysinfo.CPUStat) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CPUstats = append(c.CPUstats, stats...)
	return len(c.CPUstats)
}

func (c *Client) AddCPUStatsFB(stats []byte) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CPUstatsFB = append(c.CPUstatsFB, stats)
	return len(c.CPUstats)
}

// TODO: the current stats get copied for send; should the stat slice
// get reset now?  There is possible data loss this way; but there's
// possible data loss if the stat slice gets appended to between this
// copy op and the send completing, unless a lock is held the entire time,
// which could block the stats reading process leading to data loss due to
// missed reads.  I'm thinking COW or delete now; but punting becasue it
// really doesn't matter as this is just an experiment, right now.
// This also applies to CPUStatsFB
func (c *Client) CPUStats() []sysinfo.CPUStat {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats := make([]sysinfo.CPUStat, len(c.CPUstats))
	copy(stats, c.CPUstats)
	return stats
}

// If the message send fails, whatever was cached will be lost.
// TODO: should the messages to be sent be copied to a send cache so that
// there isn't data loss on a failed send?  Consecutive PushPeriods that
// failed to send may be problematic in that situation.
func (c *Client) CPUStatsFB() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats := make([][]byte, len(c.CPUstatsFB))
	copy(stats, c.CPUstatsFB)
	c.CPUstatsFB = nil
	return stats
}

func (c *Client) HealthBeat() {
	if c.HealthBeatPeriod == 0 {
		return
	}
	cpuCh := make(chan []sysinfo.CPUStat)
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
			fmt.Println("cpu stats read")
			c.AddCPUStats(stats)
		case <-t.C:
			fmt.Println("send cpu stats")
			c.SendCPUStats()
		}
	}
done:
	// Flush the buffer.
	c.SendCPUStats()
}

func (c *Client) HealthBeatFB() {
	fmt.Println("started HealthBeatFB")
	if c.HealthBeatPeriod == 0 {
		return
	}
	cpuCh := make(chan []byte)
	go sysinfo.CPUStatsFBTicker(c.HealthBeatPeriod, cpuCh)
	t := time.NewTicker(c.PushPeriod)
	defer t.Stop()
	for {
		select {
		case stats, ok := <-cpuCh:
			if !ok {
				fmt.Println("fb: cpu stats chan closed")
				goto done
			}
			c.AddCPUStatsFB(stats)
		case <-t.C:
			fmt.Println("fb: time to send the cpu stats")
			c.SendCPUStatsFB()
		}
	}
done:
	// Flush the buffer.
	c.SendCPUStatsFB()
}

func (c *Client) SendCPUStats() error {
	// convert the stats to bytes
	stats := c.CPUStats()
	b, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	// create the message
	msg := message.New(c.ID)
	msg.Type = websocket.BinaryMessage
	msg.Kind = message.CPUStat
	msg.DestID = c.ServerID
	msg.Data = b
	// send
	c.Send <- msg
	// TODO: only reset the stats if the send was received by the server
	c.ResetCPUStats()
	return nil
}

func (c *Client) SendCPUStatsFB() error {
	// Get a copy of the stats
	stats := c.CPUStatsFB()
	fmt.Printf("cpustatsfb: %d messages to send\n", len(stats))
	// for each stat, send a message
	for i, stat := range stats {
		msg := message.New(c.ID)
		msg.Type = websocket.BinaryMessage
		msg.Kind = message.CPUStat
		msg.DestID = c.ServerID
		msg.Data = stat
		// send
		c.Send <- msg
		fmt.Fprintf(os.Stdout, "CPUStatsFB: messages %d sent\n", i+1)
	}
	// TODO: only reset the stats if the send was received by the server
	c.ResetCPUStatsFB()
	return nil
}

func (c *Client) SetIsServer(b bool) {
	c.isServer = b
}

func (c *Client) IsServer() bool {
	return c.isServer
}

func (c *Client) ResetCPUStats() {
	c.mu.Lock()
	c.CPUstats = nil
	c.mu.Unlock()
}

// TODO: is this obsolete now that copying the stats to the sending process
// does this?
func (c *Client) ResetCPUStatsFB() {
	c.mu.Lock()
	c.CPUstats = nil
	c.mu.Unlock()
}

func (c *Client) processBinaryMessage(p []byte) error {
	// unmarshal the message
	msg, err := message.JSONUnmarshal(p)
	if err != nil {
		return err
	}
	// process according to kind
	switch msg.Kind {
	case message.CPUStat:
		//		var stats []sysinfo.CPUStat
		//		err := json.Unmarshal(msg.Data, &stats)
		//		if err != nil {
		//			fmt.Fprintf(os.Stderr, "cpu stats unmarshal error: %s", err)
		//			return err
		//		}
		//		for _, stat := range stats {
		//			fmt.Println(stat)
		//		}
		s := sysinfo.UnmarshalCPUStatsFBToString(msg.Data)
		fmt.Println(s)
	default:
		fmt.Println(string(p[1:]))
	}
	return nil
}
