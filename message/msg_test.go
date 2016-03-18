package message

import "testing"

func TestNew(t *testing.T) {
	q := NewQueue(10)
	if q.Cap() != 10 {
		t.Errorf("expected 10, got %d", cap(q.Items))
	}
}

// tests enqueue, growth, capacity restriction, and basic dequeue
func TestQueueing(t *testing.T) {
	var tests = []struct {
		size        int
		enqueueLen  int
		enqueueHead int
		dequeueCnt  int
		dequeueLen  int
		dequeueHead int
		expectedCap int
		items       []QMessage
	}{
		{size: 2, enqueueLen: 2, enqueueHead: 0, dequeueCnt: 2, dequeueHead: 0, dequeueLen: 0, expectedCap: 2, items: []QMessage{QMessage{Generic, []byte{}}, QMessage{Command, []byte{}}}},
		{size: 2, enqueueLen: 5, enqueueHead: 0, dequeueCnt: 3, dequeueHead: 3, dequeueLen: 2, expectedCap: 8, items: []QMessage{QMessage{Generic, []byte{}}, QMessage{Command, []byte{}}, QMessage{ClientInf, []byte{}}, QMessage{ClientCfg, []byte{}}, QMessage{CPUData, []byte{}}}},
		{size: 2, enqueueLen: 4, enqueueHead: 0, dequeueCnt: 1, dequeueHead: 1, dequeueLen: 3, expectedCap: 4, items: []QMessage{QMessage{Generic, []byte{}}, QMessage{Command, []byte{}}, QMessage{ClientInf, []byte{}}, QMessage{ClientCfg, []byte{}}}},
	}
	for i, test := range tests {
		q := NewQueue(test.size)
		for _, v := range test.items {
			q.Enqueue(v)

		}
		// check that the items are as expected:
		if q.Len() != test.enqueueLen {
			t.Errorf("%d: expected %d items in queue; got %d", i, test.enqueueLen, q.Len())
			if q.Cap() != test.expectedCap {
			}
			t.Errorf("%d: expected queue cap to be %d; got %d", i, test.expectedCap, q.Cap())
		}
		// Check that all enqueued items match items to queue: queued items should
		// be in same order as they were queued.
		for j := 0; j < q.Len(); j++ {
			if q.Items[j].Kind != test.items[j].Kind {
				t.Errorf("%d: expected Kind of index %d to be %s; got %s", i, j, test.items[j].Kind.String(), q.Items[j].Kind.String())
			}
		}

		// dequeue n items and check
		for j := 0; j < test.dequeueCnt; j++ {
			next, _ := q.Dequeue()
			if next.Kind != test.items[j].Kind {
				t.Errorf("%d: expected %s; got %s", i, test.items[0].Kind, next.Kind)
				continue
			}
		}
		if q.Len() != test.dequeueLen {
			t.Errorf("%d: expected Len to be %d; got %d", i, test.dequeueLen, q.Len())
		}
		if q.Head != test.dequeueHead {
			t.Errorf("%d: expected Head pos to be %d; got %d", i, test.dequeueHead, q.Head)
		}
	}
}

func TestQIsEmpty(t *testing.T) {
	tests := []struct {
		size    int
		items   []QMessage
		isEmpty bool
	}{
		{4, []QMessage{}, true},
		{4, []QMessage{QMessage{Generic, []byte{}}, QMessage{Command, []byte{}}, QMessage{ClientInf, []byte{}}, QMessage{ClientCfg, []byte{}}}, false},
	}
	for i, test := range tests {
		q := NewQueue(test.size)
		for _, v := range test.items {
			q.Enqueue(v)
		}
		if q.IsEmpty() != test.isEmpty {
			t.Errorf("%d: expected IsEmpty() to return %t. got %t", i, test.isEmpty, q.IsEmpty())
		}
	}
}

func TestQueueResetResize(t *testing.T) {
	tests := []struct {
		size        int
		enqueue     int
		dequeue     int
		cap         int
		resize      int
		expectedLen int
		expectedCap int
	}{
		{4, 0, 0, 4, 0, 0, 4},
		{2, 2, 0, 2, 0, 2, 2},
		{2, 2, 2, 2, 0, 0, 2},
		{4, 2, 2, 4, 0, 0, 4},
		{4, 2, 0, 4, 0, 2, 4},
		{4, 1, 0, 4, 0, 1, 4},
		{4, 2, 1, 4, 0, 1, 4},
		{2, 5, 0, 8, 0, 5, 6},
		{2, 5, 5, 8, 0, 0, 2},
		{2, 5, 5, 8, 4, 0, 4},
		{2, 6, 1, 8, 0, 5, 6},
		{2, 6, 1, 8, 3, 5, 6},
		{2, 6, 1, 8, 7, 5, 7},
	}
	m := QMessage{Generic, []byte{}}
	for i, test := range tests {
		q := NewQueue(test.size)
		for j := 0; j < test.enqueue; j++ {
			q.Enqueue(m)
		}
		if q.Len() != test.enqueue {
			t.Errorf("%d: expected queue len to be %d, got %d", i, test.enqueue, q.Len())
		}
		if q.Cap() != test.cap {
			t.Errorf("%d: expected queue cap to be %d, got %d", i, test.cap, q.Cap())
		}
		for j := 0; j < test.dequeue; j++ {
			_, _ = q.Dequeue()
		}
		q.Reset()
		if q.Len() != 0 {
			t.Errorf("%d: after Reset(), expected queue len to be 0, got %d", i, q.Len())
		}
		if q.Head != 0 {
			t.Errorf("%d: after Reset(), expected queue head to be at pos 0, was at pos %d", i, q.Head)
		}
		if q.Cap() != test.cap {
			t.Errorf("%d: after Reset(), expected queue cap to be %d, got %d", i, test.cap, q.Cap())
		}
	}

	for i, test := range tests {
		q := NewQueue(test.size)
		for j := 0; j < test.enqueue; j++ {
			q.Enqueue(m)
		}
		if q.Len() != test.enqueue {
			t.Errorf("%d: expected queue len to be %d, got %d", i, test.enqueue, q.Len())
		}
		if q.Cap() != test.cap {
			t.Errorf("%d: expected queue cap to be %d, got %d", i, test.cap, q.Cap())
		}
		for j := 0; j < test.dequeue; j++ {
			_, _ = q.Dequeue()
		}
		q.Resize(test.resize)
		if q.Len() != test.expectedLen {
			t.Errorf("%d: after Resize(), expected queue len to be %d, got %d", i, test.expectedLen, q.Len())
		}
		if q.Head != 0 {
			t.Errorf("%d: after Resize(), expected queue head to be at pos 0, was at pos %d", i, q.Head)
		}
		if q.Cap() != test.expectedCap {
			t.Errorf("%d: after Resize(), expected queue cap to be %d, got %d", i, test.expectedCap, q.Cap())
		}
	}
}
