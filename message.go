package autofac

import (
	"time"

	"github.com/mohae/autofac/util"
)
// Message is a description of a communication between end-points.
type Message struct {
	// ID of the message
	ID   [16]byte
	// ID of the destination
	DestID uint32
	Type int
	Data []byte
}

func NewMessage(source uint32) Message {
	return Message{ID: newMessageID(source)}
}

// a message id consists of:
// timestamp: int64
// sourceID:  uint32
// randomBits: uint32
func newMessageID(source uint32) [16]byte {
	var id [16]byte
	tb := util.Int64ToBytes(time.Now().UnixNano())
	sid := util.Uint32ToBytes(source)
	r := util.Uint32ToBytes(util.RandUint32())
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
