package message

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/mohae/autofac/util"
)

// Message is a description of a communication between end-points.
type Message struct {
	// ID of the message
	ID [16]byte
	// ID of the destination
	DestID uint32
	// Type is the websocket message type
	Type int
	// Kind is what kind of message this is.
	Kind
	Data []byte
}

func New(source uint32) Message {
	return Message{ID: newMessageID(source)}
}

// a message id consists of:
// timestamp: int64
// sourceID:  uint32
// randomBits: uint32
func newMessageID(source uint32) [16]byte {
	var id [16]byte
	sid := make([]byte, 4)
	r := make([]byte, 4)
	tb := util.Int64ToByteSlice(time.Now().UnixNano())
	binary.LittleEndian.PutUint32(sid, source)
	binary.LittleEndian.PutUint32(r, util.RandUint32())
	id[0] = tb[0]
	id[1] = tb[1]
	id[2] = tb[2]
	id[3] = tb[3]
	id[4] = tb[4]
	id[5] = tb[5]
	id[6] = tb[6]
	id[7] = tb[7]
	id[8] = sid[0]
	id[9] = sid[1]
	id[10] = sid[2]
	id[11] = sid[3]
	id[12] = r[0]
	id[13] = r[1]
	id[14] = r[2]
	id[15] = r[3]

	return id
}

func (m *Message) JSONMarshal() ([]byte, error) {
	return json.Marshal(m)
}

func JSONUnmarshal(p []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(p, &m)
	return m, err
}
