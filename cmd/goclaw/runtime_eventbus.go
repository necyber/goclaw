package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/goclaw/goclaw/pkg/api/events"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/eventbus"
	grpcstreaming "github.com/goclaw/goclaw/pkg/grpc/streaming"
	"github.com/goclaw/goclaw/pkg/logger"
)

type lifecyclePublisher interface {
	PublishLifecycleEvent(context.Context, eventbus.LifecycleEvent) (eventbus.Envelope, error)
}

type asyncLifecyclePublisher struct {
	log       logger.Logger
	publisher lifecyclePublisher
	queue     chan eventbus.LifecycleEvent

	closeOnce sync.Once
	wg        sync.WaitGroup
}

func newAsyncLifecyclePublisher(publisher lifecyclePublisher, log logger.Logger, queueSize int) *asyncLifecyclePublisher {
	if publisher == nil {
		return nil
	}
	if queueSize <= 0 {
		queueSize = 1024
	}
	p := &asyncLifecyclePublisher{
		log:       log,
		publisher: publisher,
		queue:     make(chan eventbus.LifecycleEvent, queueSize),
	}
	p.wg.Add(1)
	go p.worker()
	return p
}

func (p *asyncLifecyclePublisher) Publish(event eventbus.LifecycleEvent) {
	if p == nil {
		return
	}
	select {
	case p.queue <- event:
	default:
		if p.log != nil {
			p.log.Warn("Dropping canonical lifecycle event due to full publish queue",
				"workflow_id", event.WorkflowID,
				"task_id", event.TaskID,
				"event_type", event.EventType,
			)
		}
	}
}

func (p *asyncLifecyclePublisher) Close() {
	if p == nil {
		return
	}
	p.closeOnce.Do(func() {
		close(p.queue)
		p.wg.Wait()
	})
}

func (p *asyncLifecyclePublisher) worker() {
	defer p.wg.Done()

	for event := range p.queue {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := p.publisher.PublishLifecycleEvent(ctx, event)
		cancel()
		if err != nil && p.log != nil {
			p.log.Warn("Canonical lifecycle event publish failed",
				"workflow_id", event.WorkflowID,
				"task_id", event.TaskID,
				"event_type", event.EventType,
				"error", err,
			)
		}
	}
}

type runtimeEventBroadcaster struct {
	web                *events.Broadcaster
	observer           *grpcstreaming.WorkflowStreamObserver
	canonicalPublisher *asyncLifecyclePublisher
}

func newRuntimeEventBroadcaster(
	web *events.Broadcaster,
	observer *grpcstreaming.WorkflowStreamObserver,
	canonicalPublisher *asyncLifecyclePublisher,
) *runtimeEventBroadcaster {
	return &runtimeEventBroadcaster{
		web:                web,
		observer:           observer,
		canonicalPublisher: canonicalPublisher,
	}
}

func (b *runtimeEventBroadcaster) BroadcastWorkflowStateChanged(workflowID, name, oldState, newState string, updatedAt time.Time) {
	if b.web != nil {
		b.web.BroadcastWorkflowStateChanged(workflowID, name, oldState, newState, updatedAt)
	}
	if b.observer != nil {
		b.observer.OnWorkflowEvent(engine.WorkflowEvent{
			WorkflowID: workflowID,
			EventType:  mapWorkflowEventType(newState),
			Status:     strings.ToUpper(newState),
			Message:    "workflow state changed",
			Timestamp:  updatedAt.Unix(),
		})
	}
	if b.canonicalPublisher != nil {
		b.canonicalPublisher.Publish(eventbus.LifecycleEvent{
			Domain:      eventbus.DomainWorkflow,
			EventType:   normalizeLifecycleEventType(newState),
			ShardKey:    workflowID,
			WorkflowID:  workflowID,
			Schema:      eventbus.SchemaVersionV1,
			OrderingKey: workflowID,
			Payload: map[string]any{
				"workflow_id": workflowID,
				"name":        name,
				"old_state":   oldState,
				"new_state":   newState,
				"status":      strings.ToUpper(newState),
				"message":     "workflow state changed",
				"timestamp":   updatedAt.Unix(),
			},
		})
	}
}

func (b *runtimeEventBroadcaster) BroadcastTaskStateChanged(
	workflowID, taskID, taskName, oldState, newState, errorMessage string,
	result any,
	updatedAt time.Time,
) {
	if b.web != nil {
		b.web.BroadcastTaskStateChanged(workflowID, taskID, taskName, oldState, newState, errorMessage, result, updatedAt)
	}
	message := "task state changed"
	if errorMessage != "" {
		message = errorMessage
	}
	if b.observer != nil {
		b.observer.OnTaskEvent(engine.TaskEvent{
			WorkflowID: workflowID,
			TaskID:     taskID,
			EventType:  mapTaskEventType(newState),
			Status:     strings.ToUpper(newState),
			Message:    message,
			Timestamp:  updatedAt.Unix(),
		})
	}
	if b.canonicalPublisher != nil {
		payload := map[string]any{
			"workflow_id": workflowID,
			"task_id":     taskID,
			"task_name":   taskName,
			"old_state":   oldState,
			"new_state":   newState,
			"status":      strings.ToUpper(newState),
			"message":     message,
			"timestamp":   updatedAt.Unix(),
		}
		if result != nil {
			payload["result"] = result
		}
		b.canonicalPublisher.Publish(eventbus.LifecycleEvent{
			Domain:      eventbus.DomainTask,
			EventType:   normalizeLifecycleEventType(newState),
			ShardKey:    workflowID,
			WorkflowID:  workflowID,
			TaskID:      taskID,
			Schema:      eventbus.SchemaVersionV1,
			OrderingKey: workflowID,
			Payload:     payload,
		})
	}
}

func normalizeLifecycleEventType(state string) string {
	value := strings.TrimSpace(strings.ToLower(state))
	if value == "" {
		return "unknown"
	}
	return value
}

func mapWorkflowEventType(state string) engine.WorkflowEventType {
	switch strings.ToLower(state) {
	case "pending":
		return engine.WorkflowEventSubmitted
	case "running":
		return engine.WorkflowEventStarted
	case "completed":
		return engine.WorkflowEventCompleted
	case "failed":
		return engine.WorkflowEventFailed
	case "cancelled":
		return engine.WorkflowEventCancelled
	default:
		return engine.WorkflowEventStarted
	}
}

func mapTaskEventType(state string) engine.TaskEventType {
	switch strings.ToLower(state) {
	case "running":
		return engine.TaskEventStarted
	case "completed":
		return engine.TaskEventCompleted
	case "failed":
		return engine.TaskEventFailed
	case "cancelled":
		return engine.TaskEventCancelled
	default:
		return engine.TaskEventProgress
	}
}
