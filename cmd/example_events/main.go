// Example program demonstrating the plugin event bus system
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dotcommander/orc/pkg/plugin"
)

func main() {
	// Create a simple logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	fmt.Println("🚀 Plugin Event Bus System Demo")
	fmt.Println("================================")

	// Create event bus
	bus := plugin.NewEventBus(logger)
	defer bus.Stop()

	ctx := context.Background()

	// Demonstrate basic publish/subscribe
	fmt.Println("\n📡 Basic Event Publishing & Subscription")
	fmt.Println("-----------------------------------------")

	// Subscribe to phase events
	sub1, err := bus.Subscribe("phase.*", func(ctx context.Context, event plugin.Event) error {
		fmt.Printf("📋 Phase Event: %s from %s\n", event.Type, event.Source)
		return nil
	}, plugin.SubscriptionOptions{
		Priority: 10,
		Async:    false,
	})
	if err != nil {
		logger.Error("Failed to subscribe to phase events", "error", err)
		return
	}

	// Subscribe to all events with lower priority
	sub2, err := bus.Subscribe(".*", func(ctx context.Context, event plugin.Event) error {
		fmt.Printf("🌐 All Events Monitor: %s\n", event.Type)
		return nil
	}, plugin.SubscriptionOptions{
		Priority: 1, // Lower priority
		Async:    true,
	})
	if err != nil {
		logger.Error("Failed to subscribe to all events", "error", err)
		return
	}

	// Publish some events
	events := []plugin.Event{
		{
			Type:   "phase.started",
			Source: "planner",
			Data:   "Planning phase started",
		},
		{
			Type:   "phase.completed",
			Source: "planner", 
			Data:   "Planning completed successfully",
		},
		{
			Type:   "system.notification",
			Source: "system",
			Data:   "System status update",
		},
	}

	for _, event := range events {
		fmt.Printf("📤 Publishing: %s\n", event.Type)
		if err := bus.Publish(ctx, event); err != nil {
			logger.Error("Failed to publish event", "error", err)
		}
		time.Sleep(100 * time.Millisecond) // Allow processing
	}

	// Demonstrate helper functions
	fmt.Println("\n🛠️  Helper Function Demo")
	fmt.Println("------------------------")

	// Using helper functions for phase events
	startEvent := plugin.NewPhaseStartedEvent("writer", "demo_session", "Write a story")
	completedEvent := plugin.NewPhaseCompletedEvent("writer", "demo_session", "Story completed", 2*time.Second)
	failedEvent := plugin.NewPhaseFailedEvent("editor", "demo_session", fmt.Errorf("validation failed"), 1, 3)

	for _, event := range []plugin.Event{startEvent, completedEvent, failedEvent} {
		fmt.Printf("📤 Publishing phase event: %s\n", event.Type)
		if err := bus.Publish(ctx, event); err != nil {
			logger.Error("Failed to publish phase event", "error", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Show metrics
	fmt.Println("\n📊 Event Bus Metrics")
	fmt.Println("-------------------")

	metrics := bus.GetMetrics()
	fmt.Printf("Total Published: %d\n", metrics.TotalPublished)
	fmt.Printf("Total Delivered: %d\n", metrics.TotalDelivered)
	fmt.Printf("Total Failed: %d\n", metrics.TotalFailed)
	fmt.Printf("Last Activity: %v\n", metrics.LastActivity.Format(time.RFC3339))

	// Show subscriptions
	fmt.Println("\n🔧 Active Subscriptions")
	fmt.Println("----------------------")

	subscriptions := bus.ListSubscriptions()
	for i, sub := range subscriptions {
		fmt.Printf("%d. Pattern: %s, Priority: %d, Async: %t\n",
			i+1, sub.Pattern, sub.Priority, sub.Async)
	}

	// Clean up subscriptions
	fmt.Println("\n🧹 Cleaning up...")
	if err := bus.Unsubscribe(sub1.ID); err != nil {
		logger.Error("Failed to unsubscribe", "error", err)
	}
	if err := bus.Unsubscribe(sub2.ID); err != nil {
		logger.Error("Failed to unsubscribe", "error", err)
	}

	// Wait for async handlers to complete
	time.Sleep(200 * time.Millisecond)

	fmt.Println("\n✅ Demo completed successfully!")
	fmt.Println("\nThe event bus system provides:")
	fmt.Println("• Thread-safe publish/subscribe")
	fmt.Println("• Pattern-based event routing")
	fmt.Println("• Priority-based handler ordering")
	fmt.Println("• Async/sync execution modes")
	fmt.Println("• Comprehensive metrics")
	fmt.Println("• Error handling & recovery")
	fmt.Println("• Phase lifecycle integration")
}