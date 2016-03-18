package message

import (
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/mohae/autofact/util"
)

// NewMessageID creates a unique ID for a message and returns it as []byte.
// a message id consists of:
//   timestamp: int64
//   sourceID:  uint32
//   random bits: uint32
func NewMessageID(source uint32) []byte {
	id := make([]byte, 16)
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

// Serialize creates a flatbuffer serialized message and returns the
// bytes.
func Serialize(ID uint32, k Kind, p []byte) []byte {
	bldr := flatbuffers.NewBuilder(0)
	id := bldr.CreateByteVector(NewMessageID(ID))
	d := bldr.CreateByteVector(p)
	MessageStart(bldr)
	MessageAddID(bldr, id)
	MessageAddType(bldr, websocket.BinaryMessage)
	MessageAddKind(bldr, k.Int16())
	MessageAddData(bldr, d)
	bldr.Finish(MessageEnd(bldr))
	return bldr.Bytes[bldr.Head():]
}

// QMessage is a message wrapper for the Queue.  Data is a serialized
// Message.
type QMessage struct {
	Kind
	Data []byte
}

// Queue is a concurrent QMessage queue.
type Queue struct {
	mu      sync.Mutex
	InitCap int
	Items   []QMessage
	Head    int
}

// returns a Message Queue with an initial cap of n
func NewQueue(n int) Queue {
	return Queue{
		InitCap: n,
		Items:   make([]QMessage, 0, n),
	}
}

// Enqueue adds a message to the end of the queue.
func (q *Queue) Enqueue(m QMessage) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.Items = append(q.Items, m)
}

// Dequeue dequeues the next message in the queue.
// TODO: support shifting?
func (q *Queue) Dequeue() (QMessage, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.isEmpty() {
		return QMessage{}, false
	}
	q.Head++
	// if this dequeue results in the queue being empty. reset it
	if q.isEmpty() {
		defer q.reset()
	}
	return q.Items[q.Head-1], true
}

// IsEmpty returns whether or not the queue is empty
// IsEmpty returns whether or not the queue is empty
func (q *Queue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.isEmpty()
}

// isEmpty is an unexported version that doesn't lock because the caller
// will have handled that. Reduces multiple locks/unlocks during operations
// that need to check for emptiness and have already obtained a lock
func (q *Queue) isEmpty() bool {
	if q.Head == len(q.Items) {
		return true
	}
	return false
}

// Len returns the current number of items in the queue
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.Items) - q.Head
}

// Cap returns the current size of the queue
func (q *Queue) Cap() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return cap(q.Items)
}

// Reset resets the queue; Head and tail point to element 0. This does not
// shrink the queue; for that use Resize(). Any items in the queue will be
// lost.
func (q *Queue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.reset()
}

func (q *Queue) reset() {
	q.Head = 0
	q.Items = q.Items[:0]
}

// Resize resizes the queue to the received size, or, either its original
// capacity or to 1,25 * the number of items in the queue, whichever is larger.
// When a size of 0 is received, the queue will be set to either 1.25 * the
// number of items in the queue or its initial capacity, whichever is larger.
// Queues with space at the front are shifted to the front.
func (q *Queue) Resize(size int) int {
	q.mu.Lock()
	i := int(math.Mod(float64(len(q.Items)), float64(cap(q.Items)))*1.25) - q.Head
	if i < q.InitCap {
		i = q.InitCap
	}
	if size > i {
		i = size
	}
	tmp := make([]QMessage, 0, i)
	// if necessary, shift Items to front.
	if q.Head > 0 || len(q.Items) > 0 {
		tmp = append(tmp, q.Items[q.Head:]...)
		q.Head = 0
	}
	q.Items = tmp
	q.mu.Unlock()
	return i
}
