package engine

import (
	"sync"
)

// Job represents a single async webhook task payload
type Job struct {
	ID        int
	TargetURL string
	Payload   string
}

// MetricsTracker uses an RWMutex to track system metrics safely across multiple goroutines
type MetricsTracker struct {
	mu           sync.RWMutex
	TotalSuccess int
	TotalFailed  int
}

// IncrementSuccess updates the success metric safely via a Write Lock
func (m *MetricsTracker) IncrementSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalSuccess++
}

// GetSnapshot reads metrics concurrently via a Read Lock
func (m *MetricsTracker) GetSnapshot() (int, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.TotalSuccess, m.TotalFailed
}

// WorkerPool manages the background processing lifecycle
type WorkerPool struct {
	JobQueue    chan Job
	Metrics     *MetricsTracker
	WorkerCount int
	WG          sync.WaitGroup
}

// NewWorkerPool instantiates a thread-safe processing engine
func NewWorkerPool(workers int, queueBound int) *WorkerPool {
	return &WorkerPool{
		JobQueue:    make(chan Job, queueBound),
		Metrics:     &MetricsTracker{},
		WorkerCount: workers,
	}
}
