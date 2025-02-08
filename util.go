package mqtt

import "sync/atomic"

// a lock free, wait free, single producer, single consumer fifo queue
type Fifo[T any] struct {
	buf        []T
	pushCursor atomic.Int64
	popCursor  atomic.Int64
	capacity   int
}

func NewFifo[T any](capacity int) Fifo[T] {
	if (capacity & (capacity - 1)) != 0 {
		panic("fifo capacity must be a power of 2")
	}
	return Fifo[T]{
		buf:        make([]T, capacity),
		pushCursor: atomic.Int64{},
		popCursor:  atomic.Int64{},
		capacity:   capacity,
	}
}
func (f *Fifo[T]) Push(x T) bool {
	// TODO:
	return true
}
func (f *Fifo[T]) Pop(x *T) bool {
	// TODO:
	return true
}
