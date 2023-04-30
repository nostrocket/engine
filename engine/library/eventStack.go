package library

import (
	"github.com/nbd-wtf/go-nostr"
)

// NewEventStack returns a new Event stack (FIFO) with the given initial size.
func NewEventStack(size int) *Stack {
	return &Stack{
		nodes: make([]*nostr.Event, size),
		size:  size,
	}
}

// Stack is a FIFO stack that resizes as needed.
type Stack struct {
	nodes []*nostr.Event
	size  int
	head  int
	tail  int
	count int
}

// Push adds an Event to the stack.
func (q *Stack) Push(n *nostr.Event) {
	if q.head == q.tail && q.count > 0 {
		nodes := make([]*nostr.Event, len(q.nodes)+q.size)
		copy(nodes, q.nodes[q.head:])
		copy(nodes[len(q.nodes)-q.head:], q.nodes[:q.head])
		q.head = 0
		q.tail = len(q.nodes)
		q.nodes = nodes
	}
	q.nodes[q.tail] = n
	q.tail = (q.tail + 1) % len(q.nodes)
	q.count++
}

// Pop removes and returns an Event from the stack in first to last order.
func (q *Stack) Pop() (*nostr.Event, bool) {
	if q.count == 0 {
		return nil, false
	}
	node := q.nodes[q.head]
	q.head = (q.head + 1) % len(q.nodes)
	q.count--
	return node, true
}
