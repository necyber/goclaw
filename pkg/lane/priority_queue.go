package lane

import (
	"container/heap"
	"sync"
)

// priorityItem is an item in the priority queue.
type priorityItem struct {
	task     Task
	priority int
	seq      int64
	// index is used by heap.Interface methods.
	index int
}

// priorityHeap implements heap.Interface for priority items.
// This is an internal type used by PriorityQueue.
type priorityHeap []*priorityItem

func (h priorityHeap) Len() int { return len(h) }

func (h priorityHeap) Less(i, j int) bool {
	// Higher priority comes first (reverse order)
	if h[i].priority != h[j].priority {
		return h[i].priority > h[j].priority
	}
	// Deterministic tie-breaker: earlier enqueued task first.
	return h[i].seq < h[j].seq
}

func (h priorityHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *priorityHeap) Push(x any) {
	item := x.(*priorityItem)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *priorityHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*h = old[0 : n-1]
	return item
}

// PriorityQueue implements a priority queue using a min-heap.
// Higher priority tasks are popped first.
type PriorityQueue struct {
	heap *priorityHeap
	mu   sync.RWMutex
	seq  int64
}

// NewPriorityQueue creates a new PriorityQueue.
func NewPriorityQueue() *PriorityQueue {
	h := make(priorityHeap, 0)
	return &PriorityQueue{
		heap: &h,
	}
}

// Len returns the number of items in the queue.
func (pq *PriorityQueue) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.heap.Len()
}

// Push adds a task to the queue.
func (pq *PriorityQueue) Push(task Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &priorityItem{
		task:     task,
		priority: task.Priority(),
		seq:      pq.seq,
	}
	pq.seq++
	heap.Push(pq.heap, item)
}

// Pop removes and returns the highest priority task.
// Returns nil if the queue is empty.
func (pq *PriorityQueue) Pop() Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.heap.Len() == 0 {
		return nil
	}

	item := heap.Pop(pq.heap).(*priorityItem)
	return item.task
}

// Peek returns the highest priority task without removing it.
// Returns nil if the queue is empty.
func (pq *PriorityQueue) Peek() Task {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.heap.Len() == 0 {
		return nil
	}

	return (*pq.heap)[0].task
}

// IsEmpty returns true if the queue is empty.
func (pq *PriorityQueue) IsEmpty() bool {
	return pq.Len() == 0
}

// Clear removes all items from the queue.
func (pq *PriorityQueue) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	h := make(priorityHeap, 0)
	pq.heap = &h
}

// ConcurrentPriorityQueue is a thread-safe priority queue with blocking operations.
type ConcurrentPriorityQueue struct {
	pq       *PriorityQueue
	notEmpty chan struct{}
	closed   bool
	mu       sync.RWMutex
}

// NewConcurrentPriorityQueue creates a new ConcurrentPriorityQueue.
func NewConcurrentPriorityQueue() *ConcurrentPriorityQueue {
	return &ConcurrentPriorityQueue{
		pq:       NewPriorityQueue(),
		notEmpty: make(chan struct{}, 1),
	}
}

// Push adds a task to the queue.
func (cpq *ConcurrentPriorityQueue) Push(task Task) {
	cpq.mu.Lock()
	defer cpq.mu.Unlock()

	if cpq.closed {
		return
	}

	wasEmpty := cpq.pq.IsEmpty()
	cpq.pq.Push(task)

	// Signal if the queue was empty
	if wasEmpty {
		select {
		case cpq.notEmpty <- struct{}{}:
		default:
		}
	}
}

// Pop removes and returns the highest priority task.
// Blocks until a task is available or the queue is closed.
func (cpq *ConcurrentPriorityQueue) Pop() (Task, bool) {
	for {
		cpq.mu.Lock()
		if cpq.closed && cpq.pq.IsEmpty() {
			cpq.mu.Unlock()
			return nil, false
		}

		if !cpq.pq.IsEmpty() {
			task := cpq.pq.Pop()
			cpq.mu.Unlock()
			return task, true
		}
		cpq.mu.Unlock()

		// Wait for the queue to have items
		<-cpq.notEmpty
	}
}

// TryPop attempts to pop without blocking.
// Returns (task, true) if successful, (nil, false) if empty.
func (cpq *ConcurrentPriorityQueue) TryPop() (Task, bool) {
	cpq.mu.Lock()
	defer cpq.mu.Unlock()

	if cpq.pq.IsEmpty() {
		return nil, false
	}

	return cpq.pq.Pop(), true
}

// Len returns the number of items in the queue.
func (cpq *ConcurrentPriorityQueue) Len() int {
	cpq.mu.RLock()
	defer cpq.mu.RUnlock()
	return cpq.pq.Len()
}

// IsEmpty returns true if the queue is empty.
func (cpq *ConcurrentPriorityQueue) IsEmpty() bool {
	cpq.mu.RLock()
	defer cpq.mu.RUnlock()
	return cpq.pq.IsEmpty()
}

// Close closes the queue.
func (cpq *ConcurrentPriorityQueue) Close() {
	cpq.mu.Lock()
	defer cpq.mu.Unlock()

	if !cpq.closed {
		cpq.closed = true
		close(cpq.notEmpty)
	}
}

// IsClosed returns true if the queue is closed.
func (cpq *ConcurrentPriorityQueue) IsClosed() bool {
	cpq.mu.RLock()
	defer cpq.mu.RUnlock()
	return cpq.closed
}
