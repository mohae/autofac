// automatically generated, do not modify

package client

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type Inf struct {
	_tab flatbuffers.Table
}

func GetRootAsInf(buf []byte, offset flatbuffers.UOffsetT) *Inf {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &Inf{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *Inf) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *Inf) ID() uint32 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetUint32(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Inf) Hostname() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Inf) Region() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Inf) Zone() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Inf) DataCenter() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func InfStart(builder *flatbuffers.Builder) { builder.StartObject(5) }
func InfAddID(builder *flatbuffers.Builder, ID uint32) { builder.PrependUint32Slot(0, ID, 0) }
func InfAddHostname(builder *flatbuffers.Builder, Hostname flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(Hostname), 0) }
func InfAddRegion(builder *flatbuffers.Builder, Region flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(2, flatbuffers.UOffsetT(Region), 0) }
func InfAddZone(builder *flatbuffers.Builder, Zone flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(3, flatbuffers.UOffsetT(Zone), 0) }
func InfAddDataCenter(builder *flatbuffers.Builder, DataCenter flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(4, flatbuffers.UOffsetT(DataCenter), 0) }
func InfEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
