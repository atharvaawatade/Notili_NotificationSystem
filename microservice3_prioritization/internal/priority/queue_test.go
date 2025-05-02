package priority

import (
	"container/heap"
	"testing"
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestPriorityQueue(t *testing.T) {
	queue := NewPriorityQueue()

	// Test empty queue
	assert.Equal(t, 0, queue.Len())
	_, ok := queue.Dequeue()
	assert.False(t, ok)
	_, ok = queue.Peek()
	assert.False(t, ok)

	// Add items with different priorities
	now := time.Now()

	highPriority := &model.NotificationMessage{
		IdempotencyKey: "high-priority",
		Priority:       "high",
		MessageType:    "transactional",
		CreatedAt:      now,
	}

	mediumPriority := &model.NotificationMessage{
		IdempotencyKey: "medium-priority",
		Priority:       "normal",
		MessageType:    "transactional",
		CreatedAt:      now,
	}

	lowPriority := &model.NotificationMessage{
		IdempotencyKey: "low-priority",
		Priority:       "low",
		MessageType:    "promotional",
		CreatedAt:      now,
	}

	// Enqueue in reverse priority order
	queue.Enqueue(lowPriority, 10)
	queue.Enqueue(mediumPriority, 50)
	queue.Enqueue(highPriority, 100)

	assert.Equal(t, 3, queue.Len())

	// Peek should return highest priority without removing
	topMsg, ok := queue.Peek()
	assert.True(t, ok)
	assert.Equal(t, "high-priority", topMsg.Message.IdempotencyKey)
	assert.Equal(t, 100, topMsg.PriorityScore)
	assert.Equal(t, 3, queue.Len()) // Length shouldn't change after peek

	// Dequeue should return items in priority order
	msg1, ok := queue.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "high-priority", msg1.Message.IdempotencyKey)
	assert.Equal(t, 100, msg1.PriorityScore)

	msg2, ok := queue.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "medium-priority", msg2.Message.IdempotencyKey)
	assert.Equal(t, 50, msg2.PriorityScore)

	msg3, ok := queue.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "low-priority", msg3.Message.IdempotencyKey)
	assert.Equal(t, 10, msg3.PriorityScore)

	// Queue should be empty now
	assert.Equal(t, 0, queue.Len())
	_, ok = queue.Dequeue()
	assert.False(t, ok)
}

func TestPriorityQueueWithEqualPriorities(t *testing.T) {
	queue := NewPriorityQueue()
	
	// Create base time
	baseTime := time.Now()
	
	// Create messages with same priority but different timestamps
	samePriority1 := &model.NotificationMessage{
		IdempotencyKey: "first",
		Priority:       "normal",
		MessageType:    "transactional",
		CreatedAt:      baseTime,
	}
	
	samePriority2 := &model.NotificationMessage{
		IdempotencyKey: "second",
		Priority:       "normal",
		MessageType:    "transactional",
		CreatedAt:      baseTime.Add(10 * time.Millisecond),
	}
	
	// Create prioritized messages with explicit queuedAt times
	prioritizedMsg1 := &model.PrioritizedMessage{
		Message:       samePriority1,
		PriorityScore: 50,
		QueuedAt:      baseTime,
	}
	
	prioritizedMsg2 := &model.PrioritizedMessage{
		Message:       samePriority2,
		PriorityScore: 50,
		QueuedAt:      baseTime.Add(20 * time.Millisecond),
	}
	
	// Directly manipulate queue for testing
	queue.mu.Lock()
	heap.Push(&queue.queue, prioritizedMsg2)
	heap.Push(&queue.queue, prioritizedMsg1)
	queue.mu.Unlock()
	
	// Older message should come out first (FIFO within same priority)
	msg, ok := queue.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "first", msg.Message.IdempotencyKey)
	
	msg, ok = queue.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "second", msg.Message.IdempotencyKey)
}
