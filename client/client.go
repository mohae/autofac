package client

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohae/autofact"
	"github.com/mohae/autofact/conf"
	"github.com/mohae/autofact/message"
	cpuutil "github.com/mohae/joefriday/cpu/utilization/flat"
	netf "github.com/mohae/joefriday/net/usage/flat"
	loadf "github.com/mohae/joefriday/sysinfo/load/flat"
	memf "github.com/mohae/joefriday/sysinfo/mem/flat"
)

const IDLen = 8

// Client is anything that talks to the server.
type Client struct {
	// The Autofact Path
	AutoPath string
	// Conn holds the configuration for connecting to the server.
	conf.Conn
	// Conf holds the client configuration (how the client behaves).
	Conf *conf.Client

	// queue of healthbeat messages to be sent.
	healthbeatQ message.Queue

	// this lock is for everything except messages or other things that are
	// already threadsafe.
	mu sync.Mutex
	// The websocket connection that this client uses.
	WS *websocket.Conn
	// Channel for outbound binary messages.  The message is assumed to be a
	// websocket.Binary type
	SendB       chan []byte
	SendStr     chan string
	isConnected bool
	ServerURL   url.URL
}

func New(c conf.Conn) *Client {
	return &Client{
		Conn:        c,
		healthbeatQ: message.NewQueue(32), // this is just an arbitrary number. TODO revisit.
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
	retryEnd := start.Add(c.ConnectPeriod.Duration)
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
		time.Sleep(c.ConnectInterval.Duration)
		fmt.Printf("unable to connect to the server %s: retrying...\n", c.ServerURL.String())
	}
	// Send the ID
	err := c.WS.WriteMessage(websocket.TextMessage, []byte(c.Conn.ID))
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
			case message.ClientConf:
				c.Conf = conf.GetRootAsClient(msg.DataBytes(), 0)
				// If there's a new ID, persist it/
				if c.Conn.ID != string(c.Conf.IDBytes()) {
					c.Conn.ID = string(c.Conf.IDBytes()) // save the ID; if it was an
					err := c.Save()
					if err != nil {
						fmt.Fprintf(os.Stderr, "error while saving Conn info: %s\n", err)
						c.WS.Close()
						return false
					}
				}
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
	fmt.Printf("%s connected\n", c.Conn.ID)
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
	defer close(doneCh)
	for {
		select {
		case p, ok := <-c.SendB:
			// don't send if not connected
			if !c.IsConnected() {
				// TODO add write to db for persistence instead.
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
			// TODO does this need to handle healthbeat?
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
			fmt.Fprintln(os.Stderr, "unable to reconnect to server")
			return
		}
		switch typ {
		case websocket.TextMessage:
			fmt.Printf("textmessage: %s\n", p)
			if bytes.Equal(p, autofact.LoadAvg) {
				err = c.LoadAvg()
				if err != nil {
					fmt.Fprintf(os.Stderr, "loadavg error: %s", err)
				}
				continue
			}
			if bytes.Equal(p, autofact.AckMsg) {
				// if this is an acknowledgement message, do nothing
				// TODO: should tracking of acks, per message, for certain
				// message kinds be done?
				continue
			}
			err = c.WS.WriteMessage(websocket.TextMessage, autofact.AckMsg)
			if err != nil {
				if _, ok := err.(*websocket.CloseError); !ok {
					return
				}
				fmt.Println("reconnect from writing message: textmessage")
				connected := c.Reconnect()
				if connected {
					continue
				}
				fmt.Fprintln(os.Stderr, "unable to reconnect to server")
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
				fmt.Fprintln(os.Stderr, "unable to reconnect to server")
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
			fmt.Fprintln(os.Stderr, "unable to reconnect to server")
			return
		}
	}
}

// LoadAvg gets the current loadavg and pushes the bytes into the message queue
func (c *Client) LoadAvg() error {
	p, err := loadf.Get()
	if err != nil {
		return err
	}
	c.SendB <- message.Serialize(c.Conn.ID, message.LoadAvg, p)
	return nil
}

// IsConnected returns if the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

// Healthbeat gathers basic system stats at a given interval.
// TODO: handle doneCh stuff
// TODO: HealthBeatPushPeriod vs sending message as they are
// generated.
func (c *Client) Healthbeat() {
	// An interval of 0 means no healthbeat
	if c.Conf.HealthbeatPeriod() == 0 {
		return
	}
	// error channel
	errCh := make(chan error)
	// ticker for network usage data
	netTicker, err := netf.NewTicker(time.Duration(c.Conf.HealthbeatPeriod()))
	if err != nil {
		errCh <- err
		return
	}
	netTickr := netTicker.(*netf.Ticker)
	// make sure the resources get cleaned up
	defer netTickr.Close()
	defer netTickr.Stop()
	//	go mem.DataTicker(time.Duration(c.Conf.HealthbeatPeriod()), memCh, doneCh, errCh)
	//	go net.DataTicker(time.Duration(c.Conf.HealthbeatPeriod()), netdevCh, doneCh, errCh)
	t := time.NewTicker(time.Duration(c.Conf.HealthbeatPushPeriod()))
	defer t.Stop()
	for {
		select {
		case data, ok := <-netTickr.Data:
			if !ok {
				fmt.Println("network usage stats chan closed")
				goto done
			}
			c.healthbeatQ.Enqueue(message.QMessage{message.NetUsage, data})
		case err := <-netTickr.Errs:
			fmt.Fprintln(os.Stderr, err)
		case <-t.C:
			c.SendHealthbeatMessages()
		}
	}
done:
	// Flush the queue.
	c.SendHealthbeatMessages()
}

func (c *Client) CPUUtilization(doneCh chan struct{}) {
	// An interval of 0 means don't collect meminfo
	if c.Conf.CPUUtilizationPeriod() == 0 {
		return
	}
	// ticker for cpu utilization data
	cpuTicker, err := cpuutil.NewTicker(time.Duration(c.Conf.HealthbeatPeriod()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "CPUUtilization: error creating ticker: %s", err)
		return
	}
	cpuTickr := cpuTicker.(*cpuutil.Ticker)
	// make sure the resources get cleaned up
	defer cpuTickr.Close()
	defer cpuTickr.Stop()
	for {
		select {
		case data, ok := <-cpuTickr.Data:
			if !ok {
				fmt.Println("CPUUtilization ticker closed")
				return
			}
			c.healthbeatQ.Enqueue(message.QMessage{message.CPUUtilization, data})
		case <-doneCh:
			return
		}
	}
}

func (c *Client) MemInfo(doneCh chan struct{}) {
	// An interval of 0 means don't collect meminfo
	if c.Conf.MemInfoPeriod() == 0 {
		return
	}
	// ticker for meminfo data
	memTicker, err := memf.NewTicker(time.Duration(c.Conf.HealthbeatPeriod()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating ticker for MemInfo: %s", err)
		return
	}
	memTickr := memTicker.(*memf.Ticker)
	defer memTickr.Close()
	defer memTickr.Stop()
	for {
		select {
		case data, ok := <-memTickr.Data:
			if !ok {
				fmt.Println("mem info chan closed")
				return
			}
			c.healthbeatQ.Enqueue(message.QMessage{message.MemInfo, data})
		case <-doneCh:
			return
		}
	}
}

// SendHealthbeatMessages sends everything in the healthbeat queue.
// TODO:  should this be more resilient?
func (c *Client) SendHealthbeatMessages() error {
	// for each data, send a message
	for {
		m, ok := c.healthbeatQ.Dequeue()
		if !ok { // nothing left to send
			break
		}
		c.SendB <- message.Serialize(c.Conn.ID, m.Kind, m.Data)
	}
	return nil
}

// SendMessage sends a single serialized message of type Kind.
func (c *Client) SendMessage(kind message.Kind, p []byte) {
	c.SendB <- message.Serialize(c.Conn.ID, kind, p)
}

// binary messages are expected to be flatbuffer encoding of message.Message.
func (c *Client) processBinaryMessage(p []byte) error {
	// unmarshal the message
	msg := message.GetRootAsMessage(p, 0)
	// process according to kind
	k := message.Kind(msg.Kind())
	switch k {
	case message.ClientConf:
		c.Conf.Deserialize(msg.DataBytes())
	default:
		fmt.Println("unknown message kind")
		fmt.Println(string(p))
	}
	return nil
}
