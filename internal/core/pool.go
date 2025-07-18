package core

import (
	"context"
	"runtime"
	"sync"
)

// WorkerPool provides optimized concurrent execution for phases
type WorkerPool[T any] struct {
	workers    int
	jobs       chan func() T
	results    chan T
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	once       sync.Once
}

// NewWorkerPool creates an optimized worker pool with CPU-aware sizing
func NewWorkerPool[T any](ctx context.Context, workers int) *WorkerPool[T] {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	
	poolCtx, cancel := context.WithCancel(ctx)
	
	pool := &WorkerPool[T]{
		workers: workers,
		jobs:    make(chan func() T, workers*2), // Buffered for better throughput
		results: make(chan T, workers*2),
		ctx:     poolCtx,
		cancel:  cancel,
	}
	
	pool.start()
	return pool
}

// start initializes worker goroutines
func (p *WorkerPool[T]) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case job, ok := <-p.jobs:
					if !ok {
						return
					}
					result := job()
					select {
					case p.results <- result:
					case <-p.ctx.Done():
						return
					}
				case <-p.ctx.Done():
					return
				}
			}
		}()
	}
}

// Submit adds a job to the pool
func (p *WorkerPool[T]) Submit(job func() T) {
	select {
	case p.jobs <- job:
	case <-p.ctx.Done():
		// Pool is shutting down
	}
}

// Results returns the results channel
func (p *WorkerPool[T]) Results() <-chan T {
	return p.results
}

// Close gracefully shuts down the pool
func (p *WorkerPool[T]) Close() {
	p.once.Do(func() {
		close(p.jobs)
		p.wg.Wait()
		close(p.results)
		p.cancel()
	})
}

// ParallelExecutor provides high-performance parallel execution for phases
type ParallelExecutor struct {
	maxConcurrency int
	pool           *WorkerPool[any]
}

// NewParallelExecutor creates an optimized parallel executor
func NewParallelExecutor(ctx context.Context, maxConcurrency int) *ParallelExecutor {
	if maxConcurrency <= 0 {
		maxConcurrency = runtime.NumCPU() * 2
	}
	
	return &ParallelExecutor{
		maxConcurrency: maxConcurrency,
		pool:           NewWorkerPool[any](ctx, maxConcurrency),
	}
}

// ExecutePhases runs multiple phases concurrently with optimal resource usage
func (e *ParallelExecutor) ExecutePhases(ctx context.Context, phases []Phase, input PhaseInput) ([]PhaseOutput, error) {
	if len(phases) == 0 {
		return nil, nil
	}
	
	// For small numbers of phases, execute sequentially to avoid overhead
	if len(phases) <= 2 {
		return e.executeSequential(ctx, phases, input)
	}
	
	return e.executeConcurrent(ctx, phases, input)
}

// executeSequential handles small phase counts efficiently
func (e *ParallelExecutor) executeSequential(ctx context.Context, phases []Phase, input PhaseInput) ([]PhaseOutput, error) {
	outputs := make([]PhaseOutput, len(phases))
	
	for i, phase := range phases {
		output, err := phase.Execute(ctx, input)
		if err != nil {
			return outputs[:i], err
		}
		outputs[i] = output
	}
	
	return outputs, nil
}

// executeConcurrent handles larger phase counts with worker pool
func (e *ParallelExecutor) executeConcurrent(ctx context.Context, phases []Phase, input PhaseInput) ([]PhaseOutput, error) {
	outputs := make([]PhaseOutput, len(phases))
	errors := make([]error, len(phases))
	
	// Submit all jobs
	for i, phase := range phases {
		idx := i
		p := phase
		e.pool.Submit(func() any {
			output, err := p.Execute(ctx, input)
			return struct {
				idx    int
				output PhaseOutput
				err    error
			}{idx, output, err}
		})
	}
	
	// Collect results
	for i := 0; i < len(phases); i++ {
		select {
		case result := <-e.pool.Results():
			res := result.(struct {
				idx    int
				output PhaseOutput
				err    error
			})
			outputs[res.idx] = res.output
			errors[res.idx] = res.err
		case <-ctx.Done():
			return outputs, ctx.Err()
		}
	}
	
	// Check for any errors
	for i, err := range errors {
		if err != nil {
			return outputs[:i], err
		}
	}
	
	return outputs, nil
}

// Close shuts down the executor
func (e *ParallelExecutor) Close() {
	if e.pool != nil {
		e.pool.Close()
	}
}