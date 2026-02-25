package client

import (
	"context"
	"fmt"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
)

// StreamingOperations provides high-level streaming operations
type StreamingOperations struct {
	client *Client
}

// Streaming returns streaming operations
func (c *Client) Streaming() *StreamingOperations {
	return &StreamingOperations{client: c}
}

// WatchWorkflow watches workflow status updates
func (s *StreamingOperations) WatchWorkflow(ctx context.Context, workflowID string, resumeFromSequence int64) (pb.StreamingService_WatchWorkflowClient, error) {
	req := &pb.WatchWorkflowRequest{
		WorkflowId:         workflowID,
		ResumeFromSequence: resumeFromSequence,
	}

	stream, err := s.client.streamingClient.WatchWorkflow(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to watch workflow: %w", err)
	}

	return stream, nil
}

// WatchTasks watches task progress updates
func (s *StreamingOperations) WatchTasks(ctx context.Context, workflowID string, taskIDs []string, terminalOnly bool) (pb.StreamingService_WatchTasksClient, error) {
	req := &pb.WatchTasksRequest{
		WorkflowId:   workflowID,
		TaskIds:      taskIDs,
		TerminalOnly: terminalOnly,
	}

	stream, err := s.client.streamingClient.WatchTasks(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to watch tasks: %w", err)
	}

	return stream, nil
}

// StreamLogs streams log entries
func (s *StreamingOperations) StreamLogs(ctx context.Context, workflowID string, minLevel pb.LogLevel, taskIDs []string) (pb.StreamingService_StreamLogsClient, error) {
	stream, err := s.client.streamingClient.StreamLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create log stream: %w", err)
	}

	// Send initial request
	req := &pb.LogStreamRequest{
		WorkflowId: workflowID,
		MinLevel:   minLevel,
		TaskIds:    taskIDs,
	}

	if err := stream.Send(req); err != nil {
		return nil, fmt.Errorf("failed to send initial request: %w", err)
	}

	return stream, nil
}

// WorkflowEventHandler is called for each workflow event
type WorkflowEventHandler func(*pb.WorkflowStatusUpdate) error

// TaskEventHandler is called for each task event
type TaskEventHandler func(*pb.TaskProgressUpdate) error

// LogEventHandler is called for each log entry
type LogEventHandler func(*pb.LogEntry) error

// WatchWorkflowWithHandler watches workflow and calls handler for each event
func (s *StreamingOperations) WatchWorkflowWithHandler(ctx context.Context, workflowID string, handler WorkflowEventHandler) error {
	stream, err := s.WatchWorkflow(ctx, workflowID, 0)
	if err != nil {
		return err
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		if err := handler(update); err != nil {
			return fmt.Errorf("handler error: %w", err)
		}

		// Stop if terminal state
		if isTerminalWorkflowStatus(update.Status) {
			return nil
		}
	}
}

// WatchTasksWithHandler watches tasks and calls handler for each event
func (s *StreamingOperations) WatchTasksWithHandler(ctx context.Context, workflowID string, taskIDs []string, handler TaskEventHandler) error {
	stream, err := s.WatchTasks(ctx, workflowID, taskIDs, false)
	if err != nil {
		return err
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		if err := handler(update); err != nil {
			return fmt.Errorf("handler error: %w", err)
		}
	}
}

// StreamLogsWithHandler streams logs and calls handler for each entry
func (s *StreamingOperations) StreamLogsWithHandler(ctx context.Context, workflowID string, minLevel pb.LogLevel, handler LogEventHandler) error {
	stream, err := s.StreamLogs(ctx, workflowID, minLevel, nil)
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		if resp.Error != nil {
			return fmt.Errorf("stream error: %s", resp.Error.Message)
		}

		for _, entry := range resp.Entries {
			if err := handler(entry); err != nil {
				return fmt.Errorf("handler error: %w", err)
			}
		}
	}
}
