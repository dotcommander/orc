package plugin

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBus_BasicPublishSubscribe(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	received := make(chan Event, 1)
	
	// Subscribe to events
	sub, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		received <- event
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Publish an event
	event := Event{
		Type:   "test.message",
		Source: "test",
		Data:   "hello world",
	}
	
	err = bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}
	
	// Verify event was received
	select {
	case receivedEvent := <-received:
		if receivedEvent.Type != event.Type {
			t.Errorf("Expected event type %s, got %s", event.Type, receivedEvent.Type)
		}
		if receivedEvent.Data != event.Data {
			t.Errorf("Expected event data %v, got %v", event.Data, receivedEvent.Data)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received within timeout")
	}
	
	// Verify subscription info
	subs := bus.ListSubscriptions()
	if len(subs) != 1 {
		t.Errorf("Expected 1 subscription, got %d", len(subs))
	}
	if subs[0].ID != sub.ID {
		t.Errorf("Subscription ID mismatch")
	}
}

func TestEventBus_PatternMatching(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	
	tests := []struct {
		pattern    string
		eventType  string
		shouldMatch bool
	}{
		{"test.*", "test.message", true},
		{"test.*", "test.another", true},
		{"test.*", "other.message", false},
		{"phase\\.(started|completed)", "phase.started", true},
		{"phase\\.(started|completed)", "phase.completed", true},
		{"phase\\.(started|completed)", "phase.failed", false},
		{".*", "any.event", true},
		{"^exact$", "exact", true},
		{"^exact$", "exact.not", false},
	}
	
	for _, test := range tests {
		received := make(chan bool, 1)
		
		sub, err := bus.Subscribe(test.pattern, func(ctx context.Context, event Event) error {
			received <- true
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to subscribe to pattern %s: %v", test.pattern, err)
		}
		
		event := Event{
			Type:   test.eventType,
			Source: "test",
			Data:   "test data",
		}
		
		err = bus.Publish(ctx, event)
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}
		
		select {
		case <-received:
			if !test.shouldMatch {
				t.Errorf("Pattern %s should not match event type %s", test.pattern, test.eventType)
			}
		case <-time.After(100 * time.Millisecond):
			if test.shouldMatch {
				t.Errorf("Pattern %s should match event type %s", test.pattern, test.eventType)
			}
		}
		
		// Clean up subscription
		bus.Unsubscribe(sub.ID)
	}
}

func TestEventBus_AsyncHandlers(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	var received int32
	var wg sync.WaitGroup
	
	// Subscribe with async handler
	_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		defer wg.Done()
		atomic.AddInt32(&received, 1)
		time.Sleep(10 * time.Millisecond) // Simulate work
		return nil
	}, SubscriptionOptions{
		Async: true,
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Publish multiple events
	numEvents := 10
	wg.Add(numEvents)
	
	for i := 0; i < numEvents; i++ {
		event := Event{
			Type:   "test.async",
			Source: "test",
			Data:   i,
		}
		
		err = bus.Publish(ctx, event)
		if err != nil {
			t.Fatalf("Failed to publish event %d: %v", i, err)
		}
	}
	
	// Wait for all handlers to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		if atomic.LoadInt32(&received) != int32(numEvents) {
			t.Errorf("Expected %d events received, got %d", numEvents, received)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Handlers did not complete within timeout")
	}
}

func TestEventBus_HandlerPriority(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	var order []int
	var mu sync.Mutex
	
	// Subscribe with different priorities
	handlers := []struct {
		priority int
		id       int
	}{
		{1, 1},
		{10, 10},
		{5, 5},
		{20, 20},
		{3, 3},
	}
	
	for _, h := range handlers {
		id := h.id
		_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
			mu.Lock()
			order = append(order, id)
			mu.Unlock()
			return nil
		}, SubscriptionOptions{
			Priority: h.priority,
			Async:    false, // Synchronous to maintain order
		})
		if err != nil {
			t.Fatalf("Failed to subscribe handler %d: %v", id, err)
		}
	}
	
	// Publish event
	event := Event{
		Type:   "test.priority",
		Source: "test",
		Data:   "priority test",
	}
	
	err := bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}
	
	// Verify order (should be descending by priority: 20, 10, 5, 3, 1)
	expected := []int{20, 10, 5, 3, 1}
	mu.Lock()
	defer mu.Unlock()
	
	if len(order) != len(expected) {
		t.Fatalf("Expected %d handlers called, got %d", len(expected), len(order))
	}
	
	for i, expectedID := range expected {
		if order[i] != expectedID {
			t.Errorf("Expected handler %d at position %d, got %d", expectedID, i, order[i])
		}
	}
}

func TestEventBus_HandlerTimeout(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	
	// Subscribe with short timeout
	_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		// Simulate long-running handler
		select {
		case <-ctx.Done():
			return ctx.Err() // Should timeout
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	}, SubscriptionOptions{
		Timeout: 50 * time.Millisecond,
		Async:   false,
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Publish event
	event := Event{
		Type:   "test.timeout",
		Source: "test",
		Data:   "timeout test",
	}
	
	start := time.Now()
	err = bus.Publish(ctx, event)
	duration := time.Since(start)
	
	// Should complete reasonably quickly due to timeout
	if duration > 150*time.Millisecond {
		t.Errorf("Publish took too long: %v", duration)
	}
	
	// Check metrics for failures
	time.Sleep(50 * time.Millisecond) // Let async cleanup finish
	metrics := bus.GetMetrics()
	if metrics.TotalFailed == 0 {
		t.Error("Expected handler timeout to be recorded as failure")
	}
}

func TestEventBus_HandlerRetry(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	var attempts int32
	
	// Subscribe with handler that fails first few times
	_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}, SubscriptionOptions{
		MaxRetries: 3,
		Async:      false,
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Publish event
	event := Event{
		Type:   "test.retry",
		Source: "test",
		Data:   "retry test",
	}
	
	err = bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}
	
	// Handler should have been called 3 times
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
	
	// Should have succeeded
	metrics := bus.GetMetrics()
	if metrics.TotalDelivered == 0 {
		t.Error("Expected successful delivery after retries")
	}
}

func TestEventBus_HandlerPanic(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	
	// Subscribe with handler that panics
	_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		panic("handler panic")
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Publish event
	event := Event{
		Type:   "test.panic",
		Source: "test",
		Data:   "panic test",
	}
	
	// Should not panic
	err = bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}
	
	// Check metrics for failures
	metrics := bus.GetMetrics()
	if metrics.TotalFailed == 0 {
		t.Error("Expected handler panic to be recorded as failure")
	}
}

func TestEventBus_FilterFunction(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	var received []Event
	var mu sync.Mutex
	
	// Subscribe with filter function
	_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
		return nil
	}, SubscriptionOptions{
		FilterFunc: func(event Event) bool {
			// Only accept events with string data
			_, ok := event.Data.(string)
			return ok
		},
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Publish events with different data types
	events := []Event{
		{Type: "test.string", Data: "string data"},
		{Type: "test.int", Data: 42},
		{Type: "test.map", Data: map[string]string{"key": "value"}},
		{Type: "test.another_string", Data: "another string"},
	}
	
	for _, event := range events {
		err = bus.Publish(ctx, event)
		if err != nil {
			t.Fatalf("Failed to publish event: %v", err)
		}
	}
	
	time.Sleep(50 * time.Millisecond) // Allow processing
	
	mu.Lock()
	defer mu.Unlock()
	
	// Should only receive string events
	if len(received) != 2 {
		t.Errorf("Expected 2 events received, got %d", len(received))
	}
	
	for _, event := range received {
		if _, ok := event.Data.(string); !ok {
			t.Errorf("Received non-string event: %v", event.Data)
		}
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	var received int32
	
	// Subscribe
	sub, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		atomic.AddInt32(&received, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Publish event
	event := Event{Type: "test.before", Data: "before unsubscribe"}
	err = bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}
	
	// Unsubscribe
	err = bus.Unsubscribe(sub.ID)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}
	
	// Publish another event
	event = Event{Type: "test.after", Data: "after unsubscribe"}
	err = bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}
	
	time.Sleep(50 * time.Millisecond) // Allow processing
	
	// Should only have received first event
	if atomic.LoadInt32(&received) != 1 {
		t.Errorf("Expected 1 event received, got %d", received)
	}
	
	// Verify subscription was removed
	subs := bus.ListSubscriptions()
	if len(subs) != 0 {
		t.Errorf("Expected 0 subscriptions after unsubscribe, got %d", len(subs))
	}
}

func TestEventBus_Metrics(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	
	// Subscribe multiple handlers
	for i := 0; i < 3; i++ {
		_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to subscribe handler %d: %v", i, err)
		}
	}
	
	// Publish events
	numEvents := 5
	for i := 0; i < numEvents; i++ {
		event := Event{
			Type:   "test.metrics",
			Source: "test",
			Data:   i,
		}
		
		err := bus.Publish(ctx, event)
		if err != nil {
			t.Fatalf("Failed to publish event %d: %v", i, err)
		}
	}
	
	time.Sleep(100 * time.Millisecond) // Allow processing
	
	// Check metrics
	metrics := bus.GetMetrics()
	
	if metrics.TotalPublished != int64(numEvents) {
		t.Errorf("Expected %d published events, got %d", numEvents, metrics.TotalPublished)
	}
	
	expectedDelivered := int64(numEvents * 3) // 3 handlers per event
	if metrics.TotalDelivered != expectedDelivered {
		t.Errorf("Expected %d delivered events, got %d", expectedDelivered, metrics.TotalDelivered)
	}
	
	if len(metrics.HandlerDurations) != 3 {
		t.Errorf("Expected 3 handler duration entries, got %d", len(metrics.HandlerDurations))
	}
	
	if metrics.LastActivity.IsZero() {
		t.Error("LastActivity should be set")
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	var received int64
	
	// Subscribe
	_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		atomic.AddInt64(&received, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Publish concurrently
	numGoroutines := 10
	eventsPerGoroutine := 100
	var wg sync.WaitGroup
	
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := Event{
					Type:   "test.concurrent",
					Source: "test",
					Data:   map[string]int{"goroutine": goroutineID, "event": j},
				}
				
				err := bus.Publish(ctx, event)
				if err != nil {
					t.Errorf("Failed to publish from goroutine %d: %v", goroutineID, err)
				}
			}
		}(i)
	}
	
	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Allow processing
	
	expectedTotal := int64(numGoroutines * eventsPerGoroutine)
	if atomic.LoadInt64(&received) != expectedTotal {
		t.Errorf("Expected %d events received, got %d", expectedTotal, received)
	}
}

func TestEventBus_Stop(t *testing.T) {
	bus := NewEventBus(slog.Default())
	
	ctx := context.Background()
	
	// Subscribe with async handler
	_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}, SubscriptionOptions{
		Async: true,
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Publish event
	event := Event{Type: "test.stop", Data: "stop test"}
	err = bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}
	
	// Stop the bus (should wait for handlers to complete)
	start := time.Now()
	bus.Stop()
	duration := time.Since(start)
	
	// Should have waited for handler to complete
	if duration < 50*time.Millisecond {
		t.Error("Stop should wait for async handlers to complete")
	}
	
	// Publishing after stop should fail
	err = bus.Publish(ctx, event)
	if err == nil {
		t.Error("Expected error when publishing to stopped bus")
	}
}

func BenchmarkEventBus_Publish(b *testing.B) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	
	// Subscribe handler
	_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		return nil
	})
	if err != nil {
		b.Fatalf("Failed to subscribe: %v", err)
	}
	
	event := Event{
		Type:   "test.benchmark",
		Source: "benchmark",
		Data:   "benchmark data",
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := bus.Publish(ctx, event)
		if err != nil {
			b.Fatalf("Failed to publish: %v", err)
		}
	}
}

func BenchmarkEventBus_PublishAsync(b *testing.B) {
	bus := NewEventBus(slog.Default())
	defer bus.Stop()
	
	ctx := context.Background()
	
	// Subscribe async handler
	_, err := bus.Subscribe("test.*", func(ctx context.Context, event Event) error {
		return nil
	}, SubscriptionOptions{
		Async: true,
	})
	if err != nil {
		b.Fatalf("Failed to subscribe: %v", err)
	}
	
	event := Event{
		Type:   "test.benchmark",
		Source: "benchmark",
		Data:   "benchmark data",
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := bus.Publish(ctx, event)
		if err != nil {
			b.Fatalf("Failed to publish: %v", err)
		}
	}
}