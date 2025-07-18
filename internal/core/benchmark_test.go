package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/dotcommander/orc/internal/core"
)

// BenchmarkOrchestratorExecution measures orchestrator performance with multiple phases
func BenchmarkOrchestratorExecution(b *testing.B) {
	storage := newMockStorage()
	
	// Create realistic phase pipeline
	phases := []core.Phase{
		&mockPhase{name: "Planning", estimatedDuration: 100 * time.Millisecond},
		&mockPhase{name: "Architecture", estimatedDuration: 200 * time.Millisecond},
		&mockPhase{name: "Implementation", estimatedDuration: 500 * time.Millisecond},
		&mockPhase{name: "Review", estimatedDuration: 100 * time.Millisecond},
	}
	
	orch := core.New(phases, storage)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := orch.Run(ctx, "benchmark test request")
		if err != nil {
			b.Fatalf("orchestrator failed: %v", err)
		}
	}
}

// BenchmarkPhaseExecution measures individual phase performance
func BenchmarkPhaseExecution(b *testing.B) {
	phase := &mockPhase{
		name: "TestPhase",
		executeFunc: func(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
			// Simulate realistic AI processing time
			time.Sleep(10 * time.Millisecond)
			return core.PhaseOutput{Data: "processed output"}, nil
		},
	}
	
	ctx := context.Background()
	input := core.PhaseInput{Request: "benchmark request"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := phase.Execute(ctx, input)
		if err != nil {
			b.Fatalf("phase failed: %v", err)
		}
	}
}

// BenchmarkValidationPerformance measures validation overhead
func BenchmarkValidationPerformance(b *testing.B) {
	phase := &mockPhase{
		name: "ValidationPhase",
		validateInputFunc: func(ctx context.Context, input core.PhaseInput) error {
			// Simulate validation logic
			if len(input.Request) < 5 {
				return nil
			}
			return nil
		},
	}
	
	ctx := context.Background()
	input := core.PhaseInput{Request: "benchmark validation request"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := phase.ValidateInput(ctx, input)
		if err != nil {
			b.Fatalf("validation failed: %v", err)
		}
	}
}

// BenchmarkStorageOperations measures storage performance
func BenchmarkStorageOperations(b *testing.B) {
	storage := newMockStorage()
	ctx := context.Background()
	testData := []byte("benchmark test data with reasonable size for realistic measurement")
	
	b.Run("Save", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := "benchmark-key-" + string(rune(i))
			err := storage.Save(ctx, key, testData)
			if err != nil {
				b.Fatalf("save failed: %v", err)
			}
		}
	})
	
	// Pre-populate for load test
	for i := 0; i < 1000; i++ {
		key := "load-test-key-" + string(rune(i))
		storage.Save(ctx, key, testData)
	}
	
	b.Run("Load", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := "load-test-key-" + string(rune(i%1000))
			_, err := storage.Load(ctx, key)
			if err != nil {
				b.Fatalf("load failed: %v", err)
			}
		}
	})
	
	b.Run("Exists", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := "load-test-key-" + string(rune(i%1000))
			exists := storage.Exists(ctx, key)
			if !exists {
				b.Fatalf("expected key to exist")
			}
		}
	})
}

// BenchmarkConcurrentPhases measures performance under concurrent load
func BenchmarkConcurrentPhases(b *testing.B) {
	storage := newMockStorage()
	
	phase := &mockPhase{
		name: "ConcurrentPhase",
		executeFunc: func(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
			// Simulate concurrent-safe processing
			time.Sleep(5 * time.Millisecond)
			return core.PhaseOutput{Data: "concurrent output"}, nil
		},
	}
	
	phases := []core.Phase{phase}
	orch := core.New(phases, storage)
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := orch.Run(ctx, "concurrent benchmark request")
			if err != nil {
				b.Errorf("concurrent execution failed: %v", err)
			}
		}
	})
}

// BenchmarkMemoryAllocation measures memory efficiency
func BenchmarkMemoryAllocation(b *testing.B) {
	storage := newMockStorage()
	
	phase := &mockPhase{
		name: "MemoryPhase",
		executeFunc: func(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
			// Create realistic data structures
			data := make(map[string]interface{})
			data["result"] = "memory test output"
			data["metadata"] = map[string]string{"processed": "true"}
			return core.PhaseOutput{Data: data}, nil
		},
	}
	
	phases := []core.Phase{phase}
	orch := core.New(phases, storage)
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := orch.Run(ctx, "memory benchmark request")
		if err != nil {
			b.Fatalf("memory test failed: %v", err)
		}
	}
}