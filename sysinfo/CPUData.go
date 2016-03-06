// automatically generated, do not modify

package sysinfo

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type CPUData struct {
	_tab flatbuffers.Table
}

func GetRootAsCPUData(buf []byte, offset flatbuffers.UOffsetT) *CPUData {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &CPUData{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *CPUData) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *CPUData) Timestamp() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) CPUID() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *CPUData) Usr() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) Nice() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) Sys() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) IOWait() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(14))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) IRQ() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(16))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) Soft() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(18))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) Steal() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(20))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) Guest() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(22))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) GNice() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(24))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUData) Idle() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(26))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func CPUDataStart(builder *flatbuffers.Builder) { builder.StartObject(12) }
func CPUDataAddTimestamp(builder *flatbuffers.Builder, Timestamp int64) { builder.PrependInt64Slot(0, Timestamp, 0) }
func CPUDataAddCPUID(builder *flatbuffers.Builder, CPUID flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(CPUID), 0) }
func CPUDataAddUsr(builder *flatbuffers.Builder, Usr int16) { builder.PrependInt16Slot(2, Usr, 0) }
func CPUDataAddNice(builder *flatbuffers.Builder, Nice int16) { builder.PrependInt16Slot(3, Nice, 0) }
func CPUDataAddSys(builder *flatbuffers.Builder, Sys int16) { builder.PrependInt16Slot(4, Sys, 0) }
func CPUDataAddIOWait(builder *flatbuffers.Builder, IOWait int16) { builder.PrependInt16Slot(5, IOWait, 0) }
func CPUDataAddIRQ(builder *flatbuffers.Builder, IRQ int16) { builder.PrependInt16Slot(6, IRQ, 0) }
func CPUDataAddSoft(builder *flatbuffers.Builder, Soft int16) { builder.PrependInt16Slot(7, Soft, 0) }
func CPUDataAddSteal(builder *flatbuffers.Builder, Steal int16) { builder.PrependInt16Slot(8, Steal, 0) }
func CPUDataAddGuest(builder *flatbuffers.Builder, Guest int16) { builder.PrependInt16Slot(9, Guest, 0) }
func CPUDataAddGNice(builder *flatbuffers.Builder, GNice int16) { builder.PrependInt16Slot(10, GNice, 0) }
func CPUDataAddIdle(builder *flatbuffers.Builder, Idle int16) { builder.PrependInt16Slot(11, Idle, 0) }
func CPUDataEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
