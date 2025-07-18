package plugin

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// TestPhaseIntegration tests the complete integration between event bus and phase execution
func TestPhaseIntegration(t *testing.T) {
	logger := slog.Default()
	bus := NewEventBus(logger)
	defer bus.Stop()

	// Create test phase
	testPhase := &mockIntegrationPhase{
		name:     "test_phase",
		duration: 50 * time.Millisecond,
	}

	// Create event-aware wrapper
	orchestrator := NewPhaseOrchestrator(bus, logger)
	wrappedPhase := orchestrator.WrapPhase(testPhase)

	// Set up event monitoring
	var eventsReceived []Event
	var eventCount int32

	_, err := bus.Subscribe("phase.*", func(ctx context.Context, event Event) error {
		eventsReceived = append(eventsReceived, event)
		atomic.AddInt32(&eventCount, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to events: %v", err)
	}

	// Execute phase
	ctx := context.Background()
	input := core.PhaseInput{
		Request:   "test request",
		SessionID: "test_session",
	}

	output, err := wrappedPhase.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Phase execution failed: %v", err)
	}

	if output.Data != "test output" {
		t.Errorf("Expected output data 'test output', got %v", output.Data)
	}

	// Wait for async event processing
	time.Sleep(100 * time.Millisecond)

	// Verify events were published
	if atomic.LoadInt32(&eventCount) < 2 {
		t.Errorf("Expected at least 2 events (started, completed), got %d", eventCount)
	}

	// Verify event types
	hasStarted := false
	hasCompleted := false

	for _, event := range eventsReceived {
		switch event.Type {
		case EventTypePhaseStarted:
			hasStarted = true
		case EventTypePhaseCompleted:
			hasCompleted = true
		}
	}

	if !hasStarted {
		t.Error("Expected phase.started event")
	}
	if !hasCompleted {
		t.Error("Expected phase.completed event")
	}
}

// TestRetryablePhaseIntegration tests the retryable phase wrapper with events
func TestRetryablePhaseIntegration(t *testing.T) {
	logger := slog.Default()
	bus := NewEventBus(logger)
	defer bus.Stop()

	// Create failing test phase
	testPhase := &mockIntegrationPhase{
		name:       "failing_phase",
		duration:   10 * time.Millisecond,
		shouldFail: true,
		failCount:  2, // Fail first 2 attempts
	}

	// Create retryable wrapper
	retryablePhase := NewRetryablePhaseWrapper(
		testPhase,
		bus,
		3,                        // max attempts
		50*time.Millisecond,      // backoff
		logger,
	)

	// Monitor events
	var eventTypes []string
	_, err := bus.Subscribe("phase.*", func(ctx context.Context, event Event) error {
		eventTypes = append(eventTypes, event.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to events: %v", err)
	}

	// Execute phase
	ctx := context.Background()
	input := core.PhaseInput{
		Request:   "test request",
		SessionID: "test_session",
	}

	output, err := retryablePhase.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Phase execution failed: %v", err)
	}

	// Wait for async event processing
	time.Sleep(200 * time.Millisecond)

	// Verify retry events were published
	hasRetrying := false
	for _, eventType := range eventTypes {
		if eventType == EventTypePhaseRetrying {
			hasRetrying = true
			break
		}
	}

	if !hasRetrying {
		t.Error("Expected phase.retrying event during retry attempts")
	}

	// Verify final success
	if output.Data != "test output" {
		t.Errorf("Expected successful output after retries, got %v", output.Data)
	}
}

// TestPhaseChainIntegration tests the phase chain orchestration with events
func TestPhaseChainIntegration(t *testing.T) {
	logger := slog.Default()
	bus := NewEventBus(logger)
	defer bus.Stop()

	// Create chain of test phases
	phases := []core.Phase{
		&mockIntegrationPhase{name: "phase1", duration: 20 * time.Millisecond},
		&mockIntegrationPhase{name: "phase2", duration: 30 * time.Millisecond},
		&mockIntegrationPhase{name: "phase3", duration: 10 * time.Millisecond},
	}

	orchestrator := NewPhaseOrchestrator(bus, logger)

	// Monitor chain events
	var chainEvents []string
	_, err := bus.Subscribe("chain.*", func(ctx context.Context, event Event) error {
		chainEvents = append(chainEvents, event.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to chain events: %v", err)
	}

	// Execute phase chain
	ctx := context.Background()
	input := core.PhaseInput{
		Request:   "chain test",
		SessionID: "chain_session",
	}

	output, err := orchestrator.ExecutePhaseChain(ctx, phases, input)
	if err != nil {
		t.Fatalf("Phase chain execution failed: %v", err)
	}

	// Wait for async event processing
	time.Sleep(100 * time.Millisecond)

	// Verify chain events
	hasChainStarted := false
	hasChainCompleted := false

	for _, eventType := range chainEvents {
		switch eventType {
		case "chain.started":
			hasChainStarted = true
		case "chain.completed":
			hasChainCompleted = true
		}
	}

	if !hasChainStarted {
		t.Error("Expected chain.started event")
	}
	if !hasChainCompleted {
		t.Error("Expected chain.completed event")
	}

	// Verify final output
	if output.Data != "test output" {
		t.Errorf("Expected chain output, got %v", output.Data)
	}
}

// TestPhaseEventSubscriber tests the convenience subscriber
func TestPhaseEventSubscriber(t *testing.T) {
	logger := slog.Default()
	bus := NewEventBus(logger)
	defer bus.Stop()

	subscriber := NewPhaseEventSubscriber(bus)

	// Track events using subscriber convenience methods
	var startedPhases []string
	var completedPhases []string
	var failedPhases []string

	// Subscribe to specific event types
	_, err := subscriber.OnPhaseStarted(func(ctx context.Context, phaseName, sessionID string, input interface{}) error {
		startedPhases = append(startedPhases, phaseName)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to phase started: %v", err)
	}

	_, err = subscriber.OnPhaseCompleted(func(ctx context.Context, phaseName, sessionID string, output interface{}, duration time.Duration) error {
		completedPhases = append(completedPhases, phaseName)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to phase completed: %v", err)
	}

	_, err = subscriber.OnPhaseFailed(func(ctx context.Context, phaseName, sessionID, errorMsg string, attempt, maxAttempts int) error {
		failedPhases = append(failedPhases, phaseName)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to phase failed: %v", err)
	}

	// Publish test events
	ctx := context.Background()

	events := []Event{
		NewPhaseStartedEvent("test_phase1", "session1", "input1"),
		NewPhaseCompletedEvent("test_phase1", "session1", "output1", 100*time.Millisecond),
		NewPhaseStartedEvent("test_phase2", "session1", "input2"),
		NewPhaseFailedEvent("test_phase2", "session1", errors.New("test error"), 1, 3),
	}

	for _, event := range events {
		if err := bus.Publish(ctx, event); err != nil {
			t.Fatalf("Failed to publish event: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify results
	if len(startedPhases) != 2 {
		t.Errorf("Expected 2 started phases, got %d: %v", len(startedPhases), startedPhases)
	}

	if len(completedPhases) != 1 {
		t.Errorf("Expected 1 completed phase, got %d: %v", len(completedPhases), completedPhases)
	}

	if len(failedPhases) != 1 {
		t.Errorf("Expected 1 failed phase, got %d: %v", len(failedPhases), failedPhases)
	}

	if startedPhases[0] != "test_phase1" || startedPhases[1] != "test_phase2" {
		t.Errorf("Unexpected started phases: %v", startedPhases)
	}

	if completedPhases[0] != "test_phase1" {
		t.Errorf("Unexpected completed phase: %v", completedPhases)
	}

	if failedPhases[0] != "test_phase2" {
		t.Errorf("Unexpected failed phase: %v", failedPhases)
	}
}

// mockIntegrationPhase implements core.Phase for testing
type mockIntegrationPhase struct {
	name       string
	duration   time.Duration
	shouldFail bool
	failCount  int
	attempts   int
}

func (m *mockIntegrationPhase) Name() string {
	return m.name
}

func (m *mockIntegrationPhase) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	m.attempts++

	// Simulate work
	time.Sleep(m.duration)

	// Handle failure logic
	if m.shouldFail && m.attempts <= m.failCount {
		return core.PhaseOutput{}, errors.New("mock phase failure")
	}

	return core.PhaseOutput{
		Data: "test output",
		Metadata: map[string]interface{}{
			"attempts": m.attempts,
		},
	}, nil
}

func (m *mockIntegrationPhase) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	return nil
}

func (m *mockIntegrationPhase) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return nil
}

func (m *mockIntegrationPhase) EstimatedDuration() time.Duration {
	return m.duration
}

func (m *mockIntegrationPhase) CanRetry(err error) bool {
	return true
}