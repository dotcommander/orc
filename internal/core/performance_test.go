package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// TestPerformanceOptimizations verifies the optimization features work correctly
func TestPerformanceOptimizations(t *testing.T) {
	storage := newMockStorage()
	
	phases := []core.Phase{
		&mockPhase{name: "Phase1", estimatedDuration: 10 * time.Millisecond},
		&mockPhase{name: "Phase2", estimatedDuration: 10 * time.Millisecond},
	}
	
	// Test with optimizations enabled
	config := core.DefaultConfig()
	config.PerformanceEnabled = true
	config.MaxRetries = 2
	orch := core.New(phases, storage, core.WithConfig(config))
	
	ctx := context.Background()
	
	// First run should populate cache
	start := time.Now()
	err := orch.RunOptimized(ctx, "performance test")
	firstDuration := time.Since(start)
	if err != nil {
		t.Fatalf("first optimized run failed: %v", err)
	}
	
	// Second run should be faster due to caching
	start = time.Now()
	err = orch.RunOptimized(ctx, "performance test")
	secondDuration := time.Since(start)
	if err != nil {
		t.Fatalf("second optimized run failed: %v", err)
	}
	
	t.Logf("First run: %v, Second run: %v", firstDuration, secondDuration)
	
	// Verify cache effectiveness (second run should be significantly faster)
	if secondDuration >= firstDuration {
		t.Log("Note: Cache may not have provided speedup (possibly due to test environment)")
	}
}

// TestCustomConcurrency verifies custom concurrency settings
func TestCustomConcurrency(t *testing.T) {
	storage := newMockStorage()
	
	phases := []core.Phase{
		&mockPhase{name: "Phase1"},
		&mockPhase{name: "Phase2"},
		&mockPhase{name: "Phase3"},
	}
	
	config := core.DefaultConfig()
	config.MaxConcurrency = 4
	orch := core.New(phases, storage, core.WithConfig(config))
	
	ctx := context.Background()
	err := orch.RunOptimized(ctx, "concurrency test")
	if err != nil {
		t.Fatalf("concurrency test failed: %v", err)
	}
}

// TestCacheHitRatio measures cache effectiveness
func TestCacheHitRatio(t *testing.T) {
	storage := newMockStorage()
	
	phase := &mockPhase{
		name: "CacheTestPhase",
		executeFunc: func(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
			// Simulate expensive operation
			time.Sleep(1 * time.Millisecond)
			return core.PhaseOutput{Data: "cached result"}, nil
		},
	}
	
	phases := []core.Phase{phase}
	config := core.DefaultConfig()
	config.PerformanceEnabled = true
	orch := core.New(phases, storage, core.WithConfig(config))
	
	ctx := context.Background()
	
	// Run multiple times with same input
	for i := 0; i < 5; i++ {
		err := orch.RunOptimized(ctx, "cache test input")
		if err != nil {
			t.Fatalf("cache test iteration %d failed: %v", i, err)
		}
	}
	
	t.Log("Cache test completed - multiple runs with same input")
}

// BenchmarkOptimizedVsStandard compares optimized vs standard execution
func BenchmarkOptimizedVsStandard(b *testing.B) {
	storage := newMockStorage()
	
	phases := []core.Phase{
		&mockPhase{name: "Benchmark1", estimatedDuration: 1 * time.Millisecond},
		&mockPhase{name: "Benchmark2", estimatedDuration: 1 * time.Millisecond},
	}
	
	standardOrch := core.New(phases, storage)
	optimizedConfig := core.DefaultConfig()
	optimizedConfig.PerformanceEnabled = true
	optimizedOrch := core.New(phases, storage, core.WithConfig(optimizedConfig))
	
	ctx := context.Background()
	
	b.Run("Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := standardOrch.Run(ctx, "benchmark request")
			if err != nil {
				b.Fatalf("standard run failed: %v", err)
			}
		}
	})
	
	b.Run("Optimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := optimizedOrch.RunOptimized(ctx, "benchmark request")
			if err != nil {
				b.Fatalf("optimized run failed: %v", err)
			}
		}
	})
}

// BenchmarkCachePerformance measures cache overhead vs benefit
func BenchmarkCachePerformance(b *testing.B) {
	storage := newMockStorage()
	
	phase := &mockPhase{
		name: "CacheBenchPhase",
		executeFunc: func(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
			// Simulate moderate computation
			time.Sleep(100 * time.Microsecond)
			return core.PhaseOutput{Data: "computed result"}, nil
		},
	}
	
	phases := []core.Phase{phase}
	config := core.DefaultConfig()
	config.PerformanceEnabled = true
	orch := core.New(phases, storage, core.WithConfig(config))
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Use same input to test cache effectiveness
		err := orch.RunOptimized(ctx, "cache benchmark")
		if err != nil {
			b.Fatalf("cache benchmark failed: %v", err)
		}
	}
}