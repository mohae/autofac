package message

import (
	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/mohae/snoflinga"
)

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
