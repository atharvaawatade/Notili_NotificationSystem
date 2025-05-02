package priority

import (
	"container/heap"
	"sync"
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/model"
)

// PriorityQueue is a thread-safe priority queue implementation
type PriorityQueue struct {
	mu    sync.RWMutex
	queue priorityHeap
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		queue: make(priorityHeap, 0),
	}
	heap.Init(&pq.queue)
	return pq
}

// Enqueue adds a message to the priority queue
func (pq *PriorityQueue) Enqueue(message *model.NotificationMessage, priorityScore int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	prioritizedMsg := &model.PrioritizedMessage{
		Message:       message,
		PriorityScore: priorityScore,
		QueuedAt:      time.Now(),
	}

	heap.Push(&pq.queue, prioritizedMsg)
}

// Dequeue removes and returns the highest priority message
func (pq *PriorityQueue) Dequeue() (*model.PrioritizedMessage, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.queue.Len() == 0 {
		return nil, false
	}

	msg := heap.Pop(&pq.queue).(*model.PrioritizedMessage)
	return msg, true
}

// Peek returns the highest priority message without removing it
func (pq *PriorityQueue) Peek() (*model.PrioritizedMessage, bool) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.queue.Len() == 0 {
		return nil, false
	}

	return pq.queue[0], true
}

// Len returns the number of messages in the queue
func (pq *PriorityQueue) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.queue.Len()
}

// priorityHeap implements the heap.Interface for priority queue
type priorityHeap []*model.PrioritizedMessage

func (h priorityHeap) Len() int { return len(h) }

// Less defines the ordering. Higher priority scores are "less" to make them come first
func (h priorityHeap) Less(i, j int) bool {
	if h[i].PriorityScore != h[j].PriorityScore {
		// Higher priority scores come first
		return h[i].PriorityScore > h[j].PriorityScore
	}
	// If priority scores are equal, older messages come first (FIFO within same priority)
	return h[i].QueuedAt.Before(h[j].QueuedAt)
}

func (h priorityHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *priorityHeap) Push(x interface{}) {
	*h = append(*h, x.(*model.PrioritizedMessage))
}

func (h *priorityHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
