// automatically generated, do not modify

package sysinfo

import (
	flatbuffers "github.com/google/flatbuffers/go"
)
type MemData struct {
	_tab flatbuffers.Table
}

func GetRootAsMemData(buf []byte, offset flatbuffers.UOffsetT) *MemData {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &MemData{}
	x.Init(buf, n + offset)
	return x
}

func (rcv *MemData) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *MemData) Timestamp() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) MemTotal() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) MemUsed() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) MemFree() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) MemShared() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) MemBuffers() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(14))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) MemCached() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(16))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) CacheUsed() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(18))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) CacheFree() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(20))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) SwapTotal() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(22))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) SwapUsed() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(24))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *MemData) SwapFree() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(26))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func MemDataStart(builder *flatbuffers.Builder) { builder.StartObject(12) }
func MemDataAddTimestamp(builder *flatbuffers.Builder, Timestamp int64) { builder.PrependInt64Slot(0, Timestamp, 0) }
func MemDataAddMemTotal(builder *flatbuffers.Builder, MemTotal int64) { builder.PrependInt64Slot(1, MemTotal, 0) }
func MemDataAddMemUsed(builder *flatbuffers.Builder, MemUsed int64) { builder.PrependInt64Slot(2, MemUsed, 0) }
func MemDataAddMemFree(builder *flatbuffers.Builder, MemFree int64) { builder.PrependInt64Slot(3, MemFree, 0) }
func MemDataAddMemShared(builder *flatbuffers.Builder, MemShared int64) { builder.PrependInt64Slot(4, MemShared, 0) }
func MemDataAddMemBuffers(builder *flatbuffers.Builder, MemBuffers int64) { builder.PrependInt64Slot(5, MemBuffers, 0) }
func MemDataAddMemCached(builder *flatbuffers.Builder, MemCached int64) { builder.PrependInt64Slot(6, MemCached, 0) }
func MemDataAddCacheUsed(builder *flatbuffers.Builder, CacheUsed int64) { builder.PrependInt64Slot(7, CacheUsed, 0) }
func MemDataAddCacheFree(builder *flatbuffers.Builder, CacheFree int64) { builder.PrependInt64Slot(8, CacheFree, 0) }
func MemDataAddSwapTotal(builder *flatbuffers.Builder, SwapTotal int64) { builder.PrependInt64Slot(9, SwapTotal, 0) }
func MemDataAddSwapUsed(builder *flatbuffers.Builder, SwapUsed int64) { builder.PrependInt64Slot(10, SwapUsed, 0) }
func MemDataAddSwapFree(builder *flatbuffers.Builder, SwapFree int64) { builder.PrependInt64Slot(11, SwapFree, 0) }
func MemDataEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT { return builder.EndObject() }
