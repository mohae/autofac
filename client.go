package autofac

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohae/autofac/message"
	"github.com/mohae/autofac/sysinfo"
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
	// Current cache for accumulated CPU Stats.
	CPUstats []sysinfo.CPUStat `json:"cpu_stats"`
	WS       *websocket.Conn   `json:"-"`
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
			err := c.WS.WriteMessage(websocket.TextMessage, []byte("message received"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing text message: %s\n", err)
				return
			}
		case websocket.BinaryMessage:
			msg, err := message.JSONUnmarshal(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error unmarshaling JSON into a message: %s\n", err)
				return
			}
			fmt.Printf("Binarymessage: %#v\n", msg)
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

func (c *Client) CPUStats() []sysinfo.CPUStat {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats := make([]sysinfo.CPUStat, len(c.CPUstats))
	copy(stats, c.CPUstats)
	return stats
}

func (c *Client) HealthBeat() {
	if c.HealthBeatPeriod == 0 {
		return
	}
	cpuCh := make(chan []sysinfo.CPUStat)
	sysinfo.CPUStatsTicker(c.HealthBeatPeriod, cpuCh)
	for {
		select {
		case stats, ok := <-cpuCh:
			if !ok {
				goto done
			}
			fmt.Println("cpu stats read")
			c.AddCPUStats(stats)
		case <-time.Tick(c.PushPeriod):
			fmt.Println("send cpu stats")
			c.SendCPUStats()
		}
	}
done:
	c.SendCPUStats()
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

func (c *Client) processBinaryMessage(p []byte) error {
	// first byte of the message lets us know what kind of message this is
	switch int(p[0]) {
	case int(message.CPUStat):
		fmt.Printf("cpustats: %s", string(p[1:]))
	default:
		fmt.Println(string(p[1:]))
	}
	return nil
}
