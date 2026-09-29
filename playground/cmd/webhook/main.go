package main

import (
	"context"
	"fmt"
	"playground/internal/engine"
	"time"
)

func main() {
	fmt.Println("=== INITIALIZING CONCURRENT WEBHOOK ENGINE ===")

	// 1. Initialize a pool of 3 workers with an ingestion queue buffer capacity of 100 slots
	pool := engine.NewWorkerPool(3, 100)

	// Create a cancelable root execution context
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Start the background worker threads
	pool.Start(rootCtx)

	// 3. Simulate high-volume ingest traffic filling up the queue channel
	fmt.Println("📨 Ingesting webhook transaction streams...")
	for i := 1; i <= 10; i++ {
		pool.JobQueue <- engine.Job{
			ID:        i,
			TargetURL: fmt.Sprintf("https://api.client-gateway.com/webhook/endpoint-%d", i),
			Payload:   `{"event":"order.completed","amount":250.00}`,
		}
	}

	// Allow the background workers to process jobs for a brief window
	time.Sleep(1 * time.Second)

	// 4. Trigger a Graceful Shutdown Sequence
	fmt.Println("🛑 Initializing graceful shutdown sequence...")

	// Trigger cascading context cancellation to all workers
	cancel()

	// Close the job queue channel to signal that no further tasks are coming
	close(pool.JobQueue)

	// Block main until all background worker loops exit completely
	fmt.Println("⏳ Waiting for active worker threads to finish processing current tasks...")
	pool.WG.Wait()

	// 5. Output thread-safe metrics snapshot records
	success, failed := pool.Metrics.GetSnapshot()
	fmt.Println("=== WEBHOOK CORE PROCESSING METRICS ===")
	fmt.Printf("Total Successful Deliveries : %d\n", success)
	fmt.Printf("Total Failed Attempts       : %d\n", failed)
	fmt.Println("✨ Webhook Engine shut down cleanly with zero goroutine leaks.")
}
