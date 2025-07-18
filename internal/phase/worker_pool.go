package phase

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// WorkItem represents a generic work item for processing
type WorkItem interface {
	ID() string
	Priority() int
}

// WorkResult represents the result of processing a work item
type WorkResult interface {
	ItemID() string
	Error() error
}

// Processor defines the function signature for processing work items
type Processor[T WorkItem, R WorkResult] func(context.Context, T) (R, error)

// WorkerPool provides concurrent processing of work items
type WorkerPool[T WorkItem, R WorkResult] struct {
	workers    int
	bufferSize int
	timeout    time.Duration
	mu         sync.RWMutex
	results    []R
}

// WorkerPoolOption allows customization of worker pool behavior
type WorkerPoolOption func(*workerPoolConfig)

type workerPoolConfig struct {
	workers    int
	bufferSize int
	timeout    time.Duration
}

// WithWorkers sets the number of concurrent workers
func WithWorkers(workers int) WorkerPoolOption {
	return func(c *workerPoolConfig) {
		if workers > 0 {
			c.workers = workers
		}
	}
}

// WithBufferSize sets the buffer size for work channels
func WithBufferSize(size int) WorkerPoolOption {
	return func(c *workerPoolConfig) {
		if size > 0 {
			c.bufferSize = size
		}
	}
}

// WithTimeout sets the timeout for individual work items
func WithTimeout(timeout time.Duration) WorkerPoolOption {
	return func(c *workerPoolConfig) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// NewWorkerPool creates a new worker pool with the specified configuration
func NewWorkerPool[T WorkItem, R WorkResult](options ...WorkerPoolOption) *WorkerPool[T, R] {
	config := workerPoolConfig{
		workers:    1,
		bufferSize: 10,
		timeout:    30 * time.Second,
	}

	for _, option := range options {
		option(&config)
	}

	return &WorkerPool[T, R]{
		workers:    config.workers,
		bufferSize: config.bufferSize,
		timeout:    config.timeout,
		results:    make([]R, 0),
	}
}

// ProcessBasic processes work items using a basic worker pool pattern
func (p *WorkerPool[T, R]) ProcessBasic(ctx context.Context, items []T, processor Processor[T, R]) ([]R, error) {
	if len(items) == 0 {
		slog.Debug("No items to process in worker pool")
		return []R{}, nil
	}

	slog.Info("Starting basic worker pool processing",
		"worker_count", p.workers,
		"item_count", len(items),
		"buffer_size", p.bufferSize,
		"timeout", p.timeout,
	)

	// Create channels for work distribution
	workCh := make(chan T, p.bufferSize)
	resultCh := make(chan R, len(items))
	errorCh := make(chan error, 1)

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		workerID := i
		slog.Debug("Starting worker",
			"worker_id", workerID,
		)
		go func(workerID int) {
			defer wg.Done()
			processedCount := 0
			for item := range workCh {
				select {
				case <-ctx.Done():
					slog.Warn("Worker cancelled",
						"worker_id", workerID,
						"processed_count", processedCount,
					)
					errorCh <- ctx.Err()
					return
				default:
					slog.Debug("Worker processing item",
						"worker_id", workerID,
						"item_id", item.ID(),
					)
					// Create a timeout context for this work item
					itemCtx, cancel := context.WithTimeout(ctx, p.timeout)
					result, err := processor(itemCtx, item)
					cancel()

					if err != nil {
						slog.Error("Worker failed to process item",
							"worker_id", workerID,
							"item_id", item.ID(),
							"error", err,
						)
						select {
						case errorCh <- fmt.Errorf("worker %d failed processing item %s: %w", workerID, item.ID(), err):
						default:
						}
						return
					}
					processedCount++
					resultCh <- result
				}
			}
			slog.Debug("Worker completed",
				"worker_id", workerID,
				"processed_count", processedCount,
			)
		}(workerID)
	}

	// Send work to workers
	slog.Debug("Distributing work to workers")
	for _, item := range items {
		workCh <- item
	}
	close(workCh)

	// Wait for all workers to complete
	wg.Wait()
	close(resultCh)
	close(errorCh)

	// Check for errors
	select {
	case err := <-errorCh:
		slog.Error("Worker pool processing failed",
			"error", err,
		)
		return nil, err
	default:
	}

	// Collect results
	var results []R
	for result := range resultCh {
		results = append(results, result)
	}

	slog.Info("Worker pool processing completed",
		"result_count", len(results),
		"expected_count", len(items),
	)

	return results, nil
}

// ProcessWithErrGroup processes work items using errgroup for better error handling
func (p *WorkerPool[T, R]) ProcessWithErrGroup(ctx context.Context, items []T, processor Processor[T, R]) ([]R, error) {
	if len(items) == 0 {
		slog.Debug("No items to process in worker pool")
		return []R{}, nil
	}

	slog.Info("Starting errgroup worker pool processing",
		"worker_count", p.workers,
		"item_count", len(items),
		"buffer_size", p.bufferSize,
		"timeout", p.timeout,
	)

	// Create a channel for distributing work
	workCh := make(chan T, p.bufferSize)

	// Use errgroup for coordinated error handling
	g, ctx := errgroup.WithContext(ctx)

	// Reset results for this processing run
	p.mu.Lock()
	p.results = make([]R, 0, len(items))
	p.mu.Unlock()

	// Start worker goroutines
	for i := 0; i < p.workers; i++ {
		workerID := i
		slog.Debug("Starting errgroup worker",
			"worker_id", workerID,
		)
		g.Go(func() error {
			processedCount := 0
			for item := range workCh {
				select {
				case <-ctx.Done():
					slog.Warn("Worker cancelled by context",
						"worker_id", workerID,
						"processed_count", processedCount,
					)
					return ctx.Err()
				default:
					slog.Debug("Worker processing item",
						"worker_id", workerID,
						"item_id", item.ID(),
						"item_priority", item.Priority(),
					)
					// Create a timeout context for this work item
					itemCtx, cancel := context.WithTimeout(ctx, p.timeout)
					result, err := processor(itemCtx, item)
					cancel()

					if err != nil {
						slog.Error("Worker failed to process item",
							"worker_id", workerID,
							"item_id", item.ID(),
							"error", err,
						)
						return fmt.Errorf("worker %d failed processing item %s: %w", workerID, item.ID(), err)
					}

					// Thread-safe result collection
					p.mu.Lock()
					p.results = append(p.results, result)
					p.mu.Unlock()
					
					processedCount++
					slog.Debug("Worker successfully processed item",
						"worker_id", workerID,
						"item_id", item.ID(),
						"processed_count", processedCount,
					)
				}
			}
			slog.Debug("Worker completed all tasks",
				"worker_id", workerID,
				"total_processed", processedCount,
			)
			return nil
		})
	}

	// Send all work items to workers
	slog.Debug("Distributing work items to workers")
	distributedCount := 0
	for _, item := range items {
		select {
		case workCh <- item:
			distributedCount++
		case <-ctx.Done():
			slog.Warn("Work distribution cancelled",
				"distributed_count", distributedCount,
				"total_items", len(items),
			)
			close(workCh)
			return nil, ctx.Err()
		}
	}
	close(workCh)
	slog.Debug("All work items distributed",
		"distributed_count", distributedCount,
	)

	// Wait for all workers to complete
	if err := g.Wait(); err != nil {
		slog.Error("Worker pool processing failed",
			"error", err,
		)
		return nil, err
	}

	// Return collected results
	p.mu.RLock()
	results := make([]R, len(p.results))
	copy(results, p.results)
	p.mu.RUnlock()

	slog.Info("Worker pool processing completed successfully",
		"result_count", len(results),
		"expected_count", len(items),
	)

	return results, nil
}

// ProcessWithSemaphore processes work items with semaphore-based concurrency control
func (p *WorkerPool[T, R]) ProcessWithSemaphore(ctx context.Context, items []T, processor Processor[T, R]) ([]R, error) {
	if len(items) == 0 {
		return []R{}, nil
	}

	g, ctx := errgroup.WithContext(ctx)

	// Use a buffered channel as a semaphore
	sem := make(chan struct{}, p.workers)

	// Reset results for this processing run
	p.mu.Lock()
	p.results = make([]R, 0, len(items))
	p.mu.Unlock()

	for _, item := range items {
		item := item // Capture loop variable

		g.Go(func() error {
			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }() // Release semaphore
			case <-ctx.Done():
				return ctx.Err()
			}

			// Process the item with timeout
			itemCtx, cancel := context.WithTimeout(ctx, p.timeout)
			defer cancel()

			result, err := processor(itemCtx, item)
			if err != nil {
				return fmt.Errorf("failed processing item %s: %w", item.ID(), err)
			}

			// Collect result
			p.mu.Lock()
			p.results = append(p.results, result)
			p.mu.Unlock()

			return nil
		})
	}

	// Wait for all goroutines
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Return collected results
	p.mu.RLock()
	results := make([]R, len(p.results))
	copy(results, p.results)
	p.mu.RUnlock()

	return results, nil
}

// ProcessBatched processes work items in batches for better memory management
func (p *WorkerPool[T, R]) ProcessBatched(ctx context.Context, items []T, batchSize int, processor Processor[T, R]) ([]R, error) {
	if len(items) == 0 {
		slog.Debug("No items to process in batched worker pool")
		return []R{}, nil
	}

	if batchSize <= 0 {
		batchSize = p.workers
	}

	totalBatches := (len(items) + batchSize - 1) / batchSize
	slog.Info("Starting batched worker pool processing",
		"total_items", len(items),
		"batch_size", batchSize,
		"total_batches", totalBatches,
	)

	var allResults []R

	// Process items in batches
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		batchNum := i/batchSize + 1

		slog.Debug("Processing batch",
			"batch_num", batchNum,
			"batch_size", len(batch),
			"total_batches", totalBatches,
		)

		// Process the batch using errgroup
		results, err := p.ProcessWithErrGroup(ctx, batch, processor)
		if err != nil {
			slog.Error("Batch processing failed",
				"batch_num", batchNum,
				"error", err,
			)
			return allResults, fmt.Errorf("batch %d failed: %w", batchNum, err)
		}

		allResults = append(allResults, results...)
		slog.Debug("Batch processed successfully",
			"batch_num", batchNum,
			"results_in_batch", len(results),
			"total_results_so_far", len(allResults),
		)
	}

	slog.Info("Batched processing completed",
		"total_results", len(allResults),
		"batches_processed", totalBatches,
	)

	return allResults, nil
}

// ProcessWithPriority processes work items with priority ordering
func (p *WorkerPool[T, R]) ProcessWithPriority(ctx context.Context, items []T, processor Processor[T, R]) ([]R, error) {
	if len(items) == 0 {
		return []R{}, nil
	}

	// Sort items by priority (higher priority first)
	sortedItems := make([]T, len(items))
	copy(sortedItems, items)
	
	// Simple insertion sort by priority
	for i := 1; i < len(sortedItems); i++ {
		key := sortedItems[i]
		j := i - 1
		for j >= 0 && sortedItems[j].Priority() < key.Priority() {
			sortedItems[j+1] = sortedItems[j]
			j--
		}
		sortedItems[j+1] = key
	}

	// Process the sorted items
	return p.ProcessWithErrGroup(ctx, sortedItems, processor)
}

// GetMetrics returns metrics about the worker pool
func (p *WorkerPool[T, R]) GetMetrics() WorkerPoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return WorkerPoolMetrics{
		Workers:        p.workers,
		BufferSize:     p.bufferSize,
		Timeout:        p.timeout,
		LastResultCount: len(p.results),
	}
}

// WorkerPoolMetrics contains metrics about worker pool performance
type WorkerPoolMetrics struct {
	Workers        int
	BufferSize     int
	Timeout        time.Duration
	LastResultCount int
}

// SimpleWorkItem provides a basic implementation of WorkItem
type SimpleWorkItem struct {
	id       string
	priority int
	data     interface{}
}

// NewSimpleWorkItem creates a new simple work item
func NewSimpleWorkItem(id string, priority int, data interface{}) *SimpleWorkItem {
	return &SimpleWorkItem{
		id:       id,
		priority: priority,
		data:     data,
	}
}

func (s *SimpleWorkItem) ID() string       { return s.id }
func (s *SimpleWorkItem) Priority() int    { return s.priority }
func (s *SimpleWorkItem) Data() interface{} { return s.data }

// SimpleWorkResult provides a basic implementation of WorkResult
type SimpleWorkResult struct {
	itemID string
	data   interface{}
	err    error
}

// NewSimpleWorkResult creates a new simple work result
func NewSimpleWorkResult(itemID string, data interface{}, err error) *SimpleWorkResult {
	return &SimpleWorkResult{
		itemID: itemID,
		data:   data,
		err:    err,
	}
}

func (s *SimpleWorkResult) ItemID() string      { return s.itemID }
func (s *SimpleWorkResult) Error() error        { return s.err }
func (s *SimpleWorkResult) Data() interface{}   { return s.data }