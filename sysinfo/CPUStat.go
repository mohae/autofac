// automatically generated, do not modify

package sysinfo

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type CPUStat struct {
	_tab flatbuffers.Table
}

func GetRootAsCPUStat(buf []byte, offset flatbuffers.UOffsetT) *CPUStat {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &CPUStat{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *CPUStat) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *CPUStat) Timestamp() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *CPUStat) CPUID() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *CPUStat) Usr() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStat) Nice() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStat) Sys() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStat) IOWait() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(14))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStat) IRQ() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(16))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStat) Soft() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(18))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStat) Steal() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(20))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStat) Guest() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(22))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStat) GNice() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(24))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *CPUStat) Idle() int16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(26))
	if o != 0 {
		return rcv._tab.GetInt16(o + rcv._tab.Pos)
	}
	return 0
}

func CPUStatStart(builder *flatbuffers.Builder) { builder.StartObject(12) }
func CPUStatAddTimestamp(builder *flatbuffers.Builder, Timestamp flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(Timestamp), 0) }
func CPUStatAddCPUID(builder *flatbuffers.Builder, CPUID flatbuffers.UOffsetT) { builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(CPUID), 0) }
func CPUStatAddUsr(builder *flatbuffers.Builder, Usr int16) { builder.PrependInt16Slot(2, Usr, 0) }
func CPUStatAddNice(builder *flatbuffers.Builder, Nice int16) { builder.PrependInt16Slot(3, Nice, 0) }
func CPUStatAddSys(builder *flatbuffers.Builder, Sys int16) { builder.PrependInt16Slot(4, Sys, 0) }
func CPUStatAddIOWait(builder *flatbuffers.Builder, IOWait int16) { builder.PrependInt16Slot(5, IOWait, 0) }
func CPUStatAddIRQ(builder *flatbuffers.Builder, IRQ int16) { builder.PrependInt16Slot(6, IRQ, 0) }
func CPUStatAddSoft(builder *flatbuffers.Builder, Soft int16) { builder.PrependInt16Slot(7, Soft, 0) }
func CPUStatAddSteal(builder *flatbuffers.Builder, Steal int16) { builder.PrependInt16Slot(8, Steal, 0) }
func CPUStatAddGuest(builder *flatbuffers.Builder, Guest int16) { builder.PrependInt16Slot(9, Guest, 0) }
func CPUStatAddGNice(builder *flatbuffers.Builder, GNice int16) { builder.PrependInt16Slot(10, GNice, 0) }
func CPUStatAddIdle(builder *flatbuffers.Builder, Idle int16) { builder.PrependInt16Slot(11, Idle, 0) }
func CPUStatEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
