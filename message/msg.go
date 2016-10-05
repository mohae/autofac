package message

import (
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/mohae/autofact/util"
	"github.com/mohae/snoflinga"
)

// NewMessageID creates a unique ID for a message and returns it as []byte.
// a message id consists of:
//   timestamp: int64
//   sourceID:  uint32
//   random bits: uint32
func NewMessageID(source string) []byte {
	id := make([]byte, 16)
	sid := make([]byte, 4)
	r := make([]byte, 4)
	tb := util.Int64ToByteSlice(time.Now().UnixNano())
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

// Serialize creates a flatbuffer serialized message and returns the
// bytes.
func Serialize(ID snoflinga.Flake, k Kind, p []byte) []byte {
	bldr := flatbuffers.NewBuilder(0)
	id := bldr.CreateByteVector(ID[:])
	d := bldr.CreateByteVector(p)
	MessageStart(bldr)
	MessageAddID(bldr, id)
	MessageAddType(bldr, websocket.BinaryMessage)
	MessageAddKind(bldr, k.Int16())
	MessageAddData(bldr, d)
	bldr.Finish(MessageEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}
