package main

import (
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"
)

type LKQueue struct {
	head   unsafe.Pointer
	tail   unsafe.Pointer
	length int64
}

type node struct {
	value interface{}
	next  unsafe.Pointer
}

func NewLKQueue() *LKQueue {
	n := unsafe.Pointer(&node{})
	return &LKQueue{
		head: n,
		tail: n,
	}
}

func (q *LKQueue) Enqueue(v interface{}) {
	defer trace("enqueue")()
	n := &node{value: v}
	for {
		tail := load(&q.tail)
		next := load(&tail.next)

		if tail == load(&q.tail) { // are tail and next consistent
			if next == nil {
				if cas(&tail.next, next, n) {
					cas(&q.tail, tail, n) //enqueue is done
					atomic.AddInt64(&q.length, 1)
					return
				}
			} else { //tail was not pointing to the last node
				//try to swing tail to the next node
				cas(&q.tail, tail, next)
			}
		}
	}
}

func (q *LKQueue) Dequeue() interface{} {
	defer trace("dequeue")()
	for {
		head := load(&q.head)
		tail := load(&q.tail)
		next := load(&head.next)
		if head == load(&q.head) { //are head,tail, and next consistent
			if head == tail { //is queue empty or tail falling behind
				if next == nil { // is queue empty?
					return nil
				}
				//tail is falling behind, try to advance it
				cas(&q.tail, tail, next)
			} else {
				// read value before cas otherwise anothor dequeue might free the next node
				v := next.value
				if cas(&q.head, head, next) {
					atomic.AddInt64(&q.length, -1)
					return v // dequeue is done, return
				}
			}
		}
	}
}

func load(p *unsafe.Pointer) (n *node) {
	return (*node)(atomic.LoadPointer(p))
}

func cas(p *unsafe.Pointer, old, new *node) bool {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}

func trace(msg string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%v spend %v\n", msg, time.Since(start))
	}
}

func main() {
	q := NewLKQueue()
	for i := 0; i < 10000; i++ {
		q.Enqueue(i)
		fmt.Println(q.length)
	}

	for j := 0; j < 10000; j++ {
		fmt.Println(q.Dequeue())
		fmt.Println(q.length)
	}
}
