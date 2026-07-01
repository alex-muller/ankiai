package wp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"golang.org/x/time/rate"
)

// Task represents a unit of work to be processed by the worker pool
type Task struct {
	ID      int
	Payload interface{}
	Process func(context.Context, interface{}) error
}

// WorkerPool manages a pool of workers with rate limiting
type WorkerPool struct {
	workers     int
	rateLimiter *rate.Limiter
	taskQueue   chan Task
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *slog.Logger
}

// NewWorkerPool creates a new worker pool with specified number of workers and rate limit
func NewWorkerPool(workers int, tasksPerMinute int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	// Convert tasks per minute to rate.Limit (tasks per second)
	rateLimit := rate.Limit(float64(tasksPerMinute) / 60.0)

	return &WorkerPool{
		workers:     workers,
		rateLimiter: rate.NewLimiter(rateLimit, tasksPerMinute), // burst allows up to tasksPerMinute
		taskQueue:   make(chan Task, workers*2),                 // buffered channel
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger.Logger.With(slog.String("component", "worker_pool")),
	}
}

// Start launches all workers
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// worker processes tasks from the queue with rate limiting
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():

			return
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}

			// Wait for rate limiter permission
			if err := wp.rateLimiter.Wait(wp.ctx); err != nil {
				continue
			}

			if err := task.Process(wp.ctx, task.Payload); err != nil {
				// TODO
			} else {
				// TODO
			}
		}
	}
}

// Submit adds a task to the worker pool queue
func (wp *WorkerPool) Submit(task Task) error {
	select {
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	case wp.taskQueue <- task:
		return nil
	}
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() {
	wp.logger.Debug("stopping worker pool")

	// Close task queue to prevent new tasks
	close(wp.taskQueue)

	// Wait for all workers to finish processing
	wp.wg.Wait()

	// Cancel context
	wp.cancel()

	wp.logger.Debug("worker pool stopped")
}
