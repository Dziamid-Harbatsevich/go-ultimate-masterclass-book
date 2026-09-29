package engine

import (
	"context"
	"fmt"
	"time"
)

// Start spins up the pool of background consuming workers
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 1; i <= wp.WorkerCount; i++ {
		wp.WG.Add(1)

		// Spawn each individual worker inside its own isolated goroutine thread
		go func(workerID int) {
			defer wp.WG.Done()
			fmt.Printf("👷 Worker [%d] launched and listening for jobs...\n", workerID)

			for {
				select {
				case <-ctx.Done(): // Intercept cascading context cancellation signal
					fmt.Printf("🛑 Worker [%d] received shutdown signal. Exiting loop.\n", workerID)
					return
				case job, o := <-wp.JobQueue:
					if !o { // Channel has been closed and fully drained
						return
					}

					// Execute the simulated webhook task
					wp.processJob(workerID, job)
				}
			}
		}(i)
	}
}

// processJob executes the network I/O work and tracks metrics safely
func (wp *WorkerPool) processJob(workerID int, j Job) {
	fmt.Printf("[Worker %d] Dispatching webhook to %s (Job ID: %d)...\n", workerID, j.TargetURL, j.ID)

	// Simulate outbound HTTP network latency
	time.Sleep(300 * time.Millisecond)

	fmt.Printf("✅ [Worker %d] Webhook transaction confirmed for URL: %s\n", workerID, j.TargetURL)
	wp.Metrics.IncrementSuccess()
}
