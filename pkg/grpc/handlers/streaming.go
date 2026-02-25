package handlers

import (
	"fmt"
	"time"

	"github.com/goclaw/goclaw/pkg/engine"
	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"github.com/goclaw/goclaw/pkg/grpc/streaming"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StreamingServiceServer implements the gRPC StreamingService
type StreamingServiceServer struct {
	pb.UnimplementedStreamingServiceServer
	registry *streaming.SubscriberRegistry
	observer *streaming.WorkflowStreamObserver
}

// NewStreamingServiceServer creates a new streaming service server
func NewStreamingServiceServer(registry *streaming.SubscriberRegistry) *StreamingServiceServer {
	return &StreamingServiceServer{
		registry: registry,
		observer: streaming.NewWorkflowStreamObserver(registry),
	}
}

// WatchWorkflow implements server-side streaming for workflow status updates
func (s *StreamingServiceServer) WatchWorkflow(req *pb.WatchWorkflowRequest, stream pb.StreamingService_WatchWorkflowServer) error {
	if req == nil || req.WorkflowId == "" {
		return status.Error(codes.InvalidArgument, "workflow_id is required")
	}

	// Subscribe to workflow events
	bufferSize := 100
	sub := s.registry.Subscribe(req.WorkflowId, bufferSize)
	defer s.registry.Unsubscribe(sub.ID)

	// Set up context cancellation
	ctx := stream.Context()

	// Send initial status update
	if err := stream.Send(&pb.WorkflowStatusUpdate{
		SequenceNumber: sub.LastSequence,
		Timestamp:      timestamppb.Now(),
		WorkflowId:     req.WorkflowId,
		Status:         pb.WorkflowStatus_WORKFLOW_STATUS_PENDING,
		Message:        "Watching workflow",
	}); err != nil {
		return status.Errorf(codes.Internal, "failed to send initial update: %v", err)
	}

	// Stream events
	for {
		select {
		case <-ctx.Done():
			return status.Error(codes.Canceled, "client disconnected")

		case err := <-sub.ErrorChan:
			return status.Errorf(codes.Internal, "stream error: %v", err)

		case event, ok := <-sub.EventChan:
			if !ok {
				return status.Error(codes.Aborted, "event channel closed")
			}

			// Convert event to proto message
			seqEvent, ok := event.(*streaming.SequencedEvent)
			if !ok {
				continue
			}

			// Skip events before resume point
			if req.ResumeFromSequence > 0 && seqEvent.Sequence <= req.ResumeFromSequence {
				continue
			}

			update, err := s.convertWorkflowEvent(seqEvent)
			if err != nil {
				continue // Skip invalid events
			}

			if err := stream.Send(update); err != nil {
				return status.Errorf(codes.Internal, "failed to send update: %v", err)
			}

			// Update last sequence
			sub.LastSequence = seqEvent.Sequence
		}
	}
}

// WatchTasks implements server-side streaming for task progress updates
func (s *StreamingServiceServer) WatchTasks(req *pb.WatchTasksRequest, stream pb.StreamingService_WatchTasksServer) error {
	if req == nil || req.WorkflowId == "" {
		return status.Error(codes.InvalidArgument, "workflow_id is required")
	}

	// Subscribe to workflow events (includes task events)
	bufferSize := 100
	sub := s.registry.Subscribe(req.WorkflowId, bufferSize)
	defer s.registry.Unsubscribe(sub.ID)

	// Set up context cancellation
	ctx := stream.Context()

	// Create task filter map
	taskFilter := make(map[string]bool)
	if len(req.TaskIds) > 0 {
		for _, taskID := range req.TaskIds {
			taskFilter[taskID] = true
		}
	}

	// Stream events
	for {
		select {
		case <-ctx.Done():
			return status.Error(codes.Canceled, "client disconnected")

		case err := <-sub.ErrorChan:
			return status.Errorf(codes.Internal, "stream error: %v", err)

		case event, ok := <-sub.EventChan:
			if !ok {
				return status.Error(codes.Aborted, "event channel closed")
			}

			// Convert event to proto message
			seqEvent, ok := event.(*streaming.SequencedEvent)
			if !ok {
				continue
			}

			// Skip events before resume point
			if req.ResumeFromSequence > 0 && seqEvent.Sequence <= req.ResumeFromSequence {
				continue
			}

			// Only process task events
			taskEvent, ok := seqEvent.Event.(engine.TaskEvent)
			if !ok {
				continue
			}

			// Apply task filter
			if len(taskFilter) > 0 && !taskFilter[taskEvent.TaskID] {
				continue
			}

			// Apply terminal-only filter
			if req.TerminalOnly && !isTerminalTaskEvent(taskEvent.EventType) {
				continue
			}

			update := s.convertTaskEvent(seqEvent.Sequence, taskEvent)
			if err := stream.Send(update); err != nil {
				return status.Errorf(codes.Internal, "failed to send update: %v", err)
			}

			// Update last sequence
			sub.LastSequence = seqEvent.Sequence
		}
	}
}

// StreamLogs implements bidirectional streaming for log entries
func (s *StreamingServiceServer) StreamLogs(stream pb.StreamingService_StreamLogsServer) error {
	ctx := stream.Context()

	// Receive initial request
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive initial request: %v", err)
	}

	if req.WorkflowId == "" {
		return status.Error(codes.InvalidArgument, "workflow_id is required")
	}

	// Subscribe to workflow events
	bufferSize := 100
	sub := s.registry.Subscribe(req.WorkflowId, bufferSize)
	defer s.registry.Unsubscribe(sub.ID)

	// Create task filter map
	taskFilter := make(map[string]bool)
	if len(req.TaskIds) > 0 {
		for _, taskID := range req.TaskIds {
			taskFilter[taskID] = true
		}
	}

	minLevel := req.MinLevel

	// Handle bidirectional streaming
	errChan := make(chan error, 1)

	// Goroutine to receive client requests (for filter updates)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				errChan <- err
				return
			}

			// Update filters
			if len(req.TaskIds) > 0 {
				taskFilter = make(map[string]bool)
				for _, taskID := range req.TaskIds {
					taskFilter[taskID] = true
				}
			}
			if req.MinLevel != pb.LogLevel_LOG_LEVEL_UNSPECIFIED {
				minLevel = req.MinLevel
			}
		}
	}()

	// Stream log entries
	for {
		select {
		case <-ctx.Done():
			return status.Error(codes.Canceled, "client disconnected")

		case err := <-errChan:
			return status.Errorf(codes.Internal, "receive error: %v", err)

		case err := <-sub.ErrorChan:
			return status.Errorf(codes.Internal, "stream error: %v", err)

		case event, ok := <-sub.EventChan:
			if !ok {
				return status.Error(codes.Aborted, "event channel closed")
			}

			// Convert event to log entries
			seqEvent, ok := event.(*streaming.SequencedEvent)
			if !ok {
				continue
			}

			logEntries := s.convertToLogEntries(seqEvent, taskFilter, minLevel)
			if len(logEntries) == 0 {
				continue
			}

			response := &pb.LogStreamResponse{
				Entries: logEntries,
			}

			if err := stream.Send(response); err != nil {
				return status.Errorf(codes.Internal, "failed to send logs: %v", err)
			}
		}
	}
}

// convertWorkflowEvent converts engine workflow event to proto message
func (s *StreamingServiceServer) convertWorkflowEvent(seqEvent *streaming.SequencedEvent) (*pb.WorkflowStatusUpdate, error) {
	workflowEvent, ok := seqEvent.Event.(engine.WorkflowEvent)
	if !ok {
		return nil, fmt.Errorf("not a workflow event")
	}

	update := &pb.WorkflowStatusUpdate{
		SequenceNumber: seqEvent.Sequence,
		Timestamp:      timestamppb.New(time.Unix(workflowEvent.Timestamp, 0)),
		WorkflowId:     workflowEvent.WorkflowID,
		Status:         convertWorkflowEventTypeToStatus(workflowEvent.EventType),
		Message:        workflowEvent.Message,
	}

	return update, nil
}

// convertTaskEvent converts engine task event to proto message
func (s *StreamingServiceServer) convertTaskEvent(sequence int64, taskEvent engine.TaskEvent) *pb.TaskProgressUpdate {
	return &pb.TaskProgressUpdate{
		SequenceNumber:  sequence,
		Timestamp:       timestamppb.New(time.Unix(taskEvent.Timestamp, 0)),
		WorkflowId:      taskEvent.WorkflowID,
		TaskId:          taskEvent.TaskID,
		Status:          convertTaskEventTypeToStatus(taskEvent.EventType),
		ProgressPercent: int32(taskEvent.Progress),
		Message:         taskEvent.Message,
	}
}

// convertToLogEntries converts events to log entries
func (s *StreamingServiceServer) convertToLogEntries(seqEvent *streaming.SequencedEvent, taskFilter map[string]bool, minLevel pb.LogLevel) []*pb.LogEntry {
	var entries []*pb.LogEntry

	switch event := seqEvent.Event.(type) {
	case engine.WorkflowEvent:
		// Skip workflow events if task filter is set (only want specific tasks)
		if len(taskFilter) > 0 {
			return nil
		}

		level := pb.LogLevel_LOG_LEVEL_INFO
		if event.EventType == engine.WorkflowEventFailed {
			level = pb.LogLevel_LOG_LEVEL_ERROR
		}

		// Apply level filter
		if minLevel != pb.LogLevel_LOG_LEVEL_UNSPECIFIED && level < minLevel {
			return nil
		}

		entries = append(entries, &pb.LogEntry{
			Timestamp:  timestamppb.New(time.Unix(event.Timestamp, 0)),
			Level:      level,
			WorkflowId: event.WorkflowID,
			Message:    fmt.Sprintf("[Workflow] %s: %s", event.Status, event.Message),
		})

	case engine.TaskEvent:
		// Apply task filter
		if len(taskFilter) > 0 && !taskFilter[event.TaskID] {
			return nil
		}

		level := pb.LogLevel_LOG_LEVEL_INFO
		if event.EventType == engine.TaskEventFailed {
			level = pb.LogLevel_LOG_LEVEL_ERROR
		}

		// Apply level filter
		if minLevel != pb.LogLevel_LOG_LEVEL_UNSPECIFIED && level < minLevel {
			return nil
		}

		entries = append(entries, &pb.LogEntry{
			Timestamp:  timestamppb.New(time.Unix(event.Timestamp, 0)),
			Level:      level,
			WorkflowId: event.WorkflowID,
			TaskId:     event.TaskID,
			Message:    fmt.Sprintf("[Task %s] %s: %s", event.TaskID, event.Status, event.Message),
		})
	}

	return entries
}

// convertWorkflowEventTypeToStatus converts workflow event type to proto status
func convertWorkflowEventTypeToStatus(eventType engine.WorkflowEventType) pb.WorkflowStatus {
	switch eventType {
	case engine.WorkflowEventSubmitted:
		return pb.WorkflowStatus_WORKFLOW_STATUS_PENDING
	case engine.WorkflowEventStarted:
		return pb.WorkflowStatus_WORKFLOW_STATUS_RUNNING
	case engine.WorkflowEventCompleted:
		return pb.WorkflowStatus_WORKFLOW_STATUS_COMPLETED
	case engine.WorkflowEventFailed:
		return pb.WorkflowStatus_WORKFLOW_STATUS_FAILED
	case engine.WorkflowEventCancelled:
		return pb.WorkflowStatus_WORKFLOW_STATUS_CANCELLED
	default:
		return pb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED
	}
}

// convertTaskEventTypeToStatus converts task event type to proto status
func convertTaskEventTypeToStatus(eventType engine.TaskEventType) pb.TaskStatus {
	switch eventType {
	case engine.TaskEventStarted:
		return pb.TaskStatus_TASK_STATUS_RUNNING
	case engine.TaskEventProgress:
		return pb.TaskStatus_TASK_STATUS_RUNNING
	case engine.TaskEventCompleted:
		return pb.TaskStatus_TASK_STATUS_COMPLETED
	case engine.TaskEventFailed:
		return pb.TaskStatus_TASK_STATUS_FAILED
	case engine.TaskEventCancelled:
		return pb.TaskStatus_TASK_STATUS_CANCELLED
	default:
		return pb.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

// isTerminalTaskEvent checks if a task event is terminal
func isTerminalTaskEvent(eventType engine.TaskEventType) bool {
	return eventType == engine.TaskEventCompleted ||
		eventType == engine.TaskEventFailed ||
		eventType == engine.TaskEventCancelled
}
