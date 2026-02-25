package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/engine"
	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"github.com/goclaw/goclaw/pkg/grpc/streaming"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// mockWatchWorkflowStream implements pb.StreamingService_WatchWorkflowServer
type mockWatchWorkflowStream struct {
	ctx     context.Context
	updates []*pb.WorkflowStatusUpdate
	sendErr error
}

func (m *mockWatchWorkflowStream) Send(update *pb.WorkflowStatusUpdate) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.updates = append(m.updates, update)
	return nil
}

func (m *mockWatchWorkflowStream) Context() context.Context {
	return m.ctx
}

func (m *mockWatchWorkflowStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockWatchWorkflowStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockWatchWorkflowStream) SetTrailer(md metadata.MD)       {}
func (m *mockWatchWorkflowStream) SendMsg(msg interface{}) error   { return nil }
func (m *mockWatchWorkflowStream) RecvMsg(msg interface{}) error   { return nil }

// mockWatchTasksStream implements pb.StreamingService_WatchTasksServer
type mockWatchTasksStream struct {
	ctx     context.Context
	updates []*pb.TaskProgressUpdate
	sendErr error
}

func (m *mockWatchTasksStream) Send(update *pb.TaskProgressUpdate) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.updates = append(m.updates, update)
	return nil
}

func (m *mockWatchTasksStream) Context() context.Context {
	return m.ctx
}

func (m *mockWatchTasksStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockWatchTasksStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockWatchTasksStream) SetTrailer(md metadata.MD)       {}
func (m *mockWatchTasksStream) SendMsg(msg interface{}) error   { return nil }
func (m *mockWatchTasksStream) RecvMsg(msg interface{}) error   { return nil }

// mockStreamLogsStream implements pb.StreamingService_StreamLogsServer
type mockStreamLogsStream struct {
	ctx       context.Context
	responses []*pb.LogStreamResponse
	requests  []*pb.LogStreamRequest
	recvIdx   int
	sendErr   error
	recvErr   error
}

func (m *mockStreamLogsStream) Send(response *pb.LogStreamResponse) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.responses = append(m.responses, response)
	return nil
}

func (m *mockStreamLogsStream) Recv() (*pb.LogStreamRequest, error) {
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	if m.recvIdx >= len(m.requests) {
		// Block forever after initial request to simulate client waiting
		select {}
	}
	req := m.requests[m.recvIdx]
	m.recvIdx++
	return req, nil
}

func (m *mockStreamLogsStream) Context() context.Context {
	return m.ctx
}

func (m *mockStreamLogsStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockStreamLogsStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockStreamLogsStream) SetTrailer(md metadata.MD)       {}
func (m *mockStreamLogsStream) SendMsg(msg interface{}) error   { return nil }
func (m *mockStreamLogsStream) RecvMsg(msg interface{}) error   { return nil }

func TestWatchWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		request     *pb.WatchWorkflowRequest
		events      []engine.WorkflowEvent
		wantErr     bool
		wantCode    codes.Code
		wantUpdates int
	}{
		{
			name:     "missing workflow_id",
			request:  &pb.WatchWorkflowRequest{},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "successful watch",
			request: &pb.WatchWorkflowRequest{
				WorkflowId: "wf-123",
			},
			events: []engine.WorkflowEvent{
				{
					WorkflowID: "wf-123",
					EventType:  engine.WorkflowEventStarted,
					Status:     "RUNNING",
					Message:    "Workflow started",
					Timestamp:  time.Now().Unix(),
				},
				{
					WorkflowID: "wf-123",
					EventType:  engine.WorkflowEventCompleted,
					Status:     "COMPLETED",
					Message:    "Workflow completed",
					Timestamp:  time.Now().Unix(),
				},
			},
			wantErr:     false,
			wantUpdates: 3, // 1 initial + 2 events
		},
		{
			name: "resume from sequence",
			request: &pb.WatchWorkflowRequest{
				WorkflowId:         "wf-123",
				ResumeFromSequence: 1,
			},
			events: []engine.WorkflowEvent{
				{
					WorkflowID: "wf-123",
					EventType:  engine.WorkflowEventStarted,
					Status:     "RUNNING",
					Message:    "Event 1",
					Timestamp:  time.Now().Unix(),
				},
				{
					WorkflowID: "wf-123",
					EventType:  engine.WorkflowEventCompleted,
					Status:     "COMPLETED",
					Message:    "Event 2",
					Timestamp:  time.Now().Unix(),
				},
			},
			wantErr:     false,
			wantUpdates: 2, // 1 initial + 1 event (skipped first)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := streaming.NewSubscriberRegistry()
			server := NewStreamingServiceServer(registry)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			stream := &mockWatchWorkflowStream{
				ctx:     ctx,
				updates: make([]*pb.WorkflowStatusUpdate, 0),
			}

			// Run WatchWorkflow in goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.WatchWorkflow(tt.request, stream)
			}()

			// Send events if workflow_id is valid
			if tt.request != nil && tt.request.WorkflowId != "" {
				time.Sleep(10 * time.Millisecond) // Let subscription happen
				for _, event := range tt.events {
					server.observer.OnWorkflowEvent(event)
					time.Sleep(5 * time.Millisecond)
				}
			}

			// Wait for completion or timeout
			err := <-errChan

			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			} else {
				// Context cancellation is expected
				if err != nil {
					st, ok := status.FromError(err)
					require.True(t, ok)
					assert.Equal(t, codes.Canceled, st.Code())
				}
				assert.GreaterOrEqual(t, len(stream.updates), tt.wantUpdates)
			}
		})
	}
}

func TestWatchTasks(t *testing.T) {
	tests := []struct {
		name        string
		request     *pb.WatchTasksRequest
		events      []engine.TaskEvent
		wantErr     bool
		wantCode    codes.Code
		wantUpdates int
	}{
		{
			name:     "missing workflow_id",
			request:  &pb.WatchTasksRequest{},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "successful watch all tasks",
			request: &pb.WatchTasksRequest{
				WorkflowId: "wf-123",
			},
			events: []engine.TaskEvent{
				{
					WorkflowID: "wf-123",
					TaskID:     "task-1",
					EventType:  engine.TaskEventStarted,
					Status:     "RUNNING",
					Progress:   0,
					Message:    "Task started",
					Timestamp:  time.Now().Unix(),
				},
				{
					WorkflowID: "wf-123",
					TaskID:     "task-2",
					EventType:  engine.TaskEventStarted,
					Status:     "RUNNING",
					Progress:   0,
					Message:    "Task started",
					Timestamp:  time.Now().Unix(),
				},
			},
			wantErr:     false,
			wantUpdates: 2,
		},
		{
			name: "watch specific tasks",
			request: &pb.WatchTasksRequest{
				WorkflowId: "wf-123",
				TaskIds:    []string{"task-1"},
			},
			events: []engine.TaskEvent{
				{
					WorkflowID: "wf-123",
					TaskID:     "task-1",
					EventType:  engine.TaskEventStarted,
					Status:     "RUNNING",
					Message:    "Task 1 started",
					Timestamp:  time.Now().Unix(),
				},
				{
					WorkflowID: "wf-123",
					TaskID:     "task-2",
					EventType:  engine.TaskEventStarted,
					Status:     "RUNNING",
					Message:    "Task 2 started",
					Timestamp:  time.Now().Unix(),
				},
			},
			wantErr:     false,
			wantUpdates: 1, // Only task-1
		},
		{
			name: "terminal only filter",
			request: &pb.WatchTasksRequest{
				WorkflowId:   "wf-123",
				TerminalOnly: true,
			},
			events: []engine.TaskEvent{
				{
					WorkflowID: "wf-123",
					TaskID:     "task-1",
					EventType:  engine.TaskEventStarted,
					Status:     "RUNNING",
					Message:    "Task started",
					Timestamp:  time.Now().Unix(),
				},
				{
					WorkflowID: "wf-123",
					TaskID:     "task-1",
					EventType:  engine.TaskEventProgress,
					Status:     "RUNNING",
					Progress:   50,
					Message:    "Task progress",
					Timestamp:  time.Now().Unix(),
				},
				{
					WorkflowID: "wf-123",
					TaskID:     "task-1",
					EventType:  engine.TaskEventCompleted,
					Status:     "COMPLETED",
					Message:    "Task completed",
					Timestamp:  time.Now().Unix(),
				},
			},
			wantErr:     false,
			wantUpdates: 1, // Only completed event
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := streaming.NewSubscriberRegistry()
			server := NewStreamingServiceServer(registry)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			stream := &mockWatchTasksStream{
				ctx:     ctx,
				updates: make([]*pb.TaskProgressUpdate, 0),
			}

			// Run WatchTasks in goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.WatchTasks(tt.request, stream)
			}()

			// Send events if workflow_id is valid
			if tt.request != nil && tt.request.WorkflowId != "" {
				time.Sleep(10 * time.Millisecond) // Let subscription happen
				for _, event := range tt.events {
					server.observer.OnTaskEvent(event)
					time.Sleep(5 * time.Millisecond)
				}
			}

			// Wait for completion or timeout
			err := <-errChan

			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			} else {
				// Context cancellation is expected
				if err != nil {
					st, ok := status.FromError(err)
					require.True(t, ok)
					assert.Equal(t, codes.Canceled, st.Code())
				}
				assert.GreaterOrEqual(t, len(stream.updates), tt.wantUpdates)
			}
		})
	}
}

func TestStreamLogs(t *testing.T) {
	tests := []struct {
		name          string
		requests      []*pb.LogStreamRequest
		events        []interface{}
		wantErr       bool
		wantCode      codes.Code
		wantResponses int
	}{
		{
			name: "missing workflow_id",
			requests: []*pb.LogStreamRequest{
				{},
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "successful log streaming",
			requests: []*pb.LogStreamRequest{
				{
					WorkflowId: "wf-123",
					MinLevel:   pb.LogLevel_LOG_LEVEL_INFO,
				},
			},
			events: []interface{}{
				engine.WorkflowEvent{
					WorkflowID: "wf-123",
					EventType:  engine.WorkflowEventStarted,
					Status:     "RUNNING",
					Message:    "Workflow started",
					Timestamp:  time.Now().Unix(),
				},
				engine.TaskEvent{
					WorkflowID: "wf-123",
					TaskID:     "task-1",
					EventType:  engine.TaskEventCompleted,
					Status:     "COMPLETED",
					Message:    "Task completed",
					Timestamp:  time.Now().Unix(),
				},
			},
			wantErr:       false,
			wantResponses: 2,
		},
		{
			name: "log level filtering",
			requests: []*pb.LogStreamRequest{
				{
					WorkflowId: "wf-123",
					MinLevel:   pb.LogLevel_LOG_LEVEL_ERROR,
				},
			},
			events: []interface{}{
				engine.WorkflowEvent{
					WorkflowID: "wf-123",
					EventType:  engine.WorkflowEventStarted,
					Status:     "RUNNING",
					Message:    "Info message",
					Timestamp:  time.Now().Unix(),
				},
				engine.TaskEvent{
					WorkflowID: "wf-123",
					TaskID:     "task-1",
					EventType:  engine.TaskEventFailed,
					Status:     "FAILED",
					Message:    "Error message",
					Timestamp:  time.Now().Unix(),
				},
			},
			wantErr:       false,
			wantResponses: 1, // Only error event
		},
		{
			name: "task filtering",
			requests: []*pb.LogStreamRequest{
				{
					WorkflowId: "wf-123",
					TaskIds:    []string{"task-1"},
					MinLevel:   pb.LogLevel_LOG_LEVEL_INFO,
				},
			},
			events: []interface{}{
				engine.TaskEvent{
					WorkflowID: "wf-123",
					TaskID:     "task-1",
					EventType:  engine.TaskEventCompleted,
					Status:     "COMPLETED",
					Message:    "Task 1 completed",
					Timestamp:  time.Now().Unix(),
				},
				engine.TaskEvent{
					WorkflowID: "wf-123",
					TaskID:     "task-2",
					EventType:  engine.TaskEventCompleted,
					Status:     "COMPLETED",
					Message:    "Task 2 completed",
					Timestamp:  time.Now().Unix(),
				},
			},
			wantErr:       false,
			wantResponses: 1, // Only task-1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := streaming.NewSubscriberRegistry()
			server := NewStreamingServiceServer(registry)

			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()

			stream := &mockStreamLogsStream{
				ctx:       ctx,
				requests:  tt.requests,
				responses: make([]*pb.LogStreamResponse, 0),
			}

			// Run StreamLogs in goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.StreamLogs(stream)
			}()

			// Send events if workflow_id is valid
			if len(tt.requests) > 0 && tt.requests[0].WorkflowId != "" {
				time.Sleep(20 * time.Millisecond) // Let subscription happen
				for _, event := range tt.events {
					switch e := event.(type) {
					case engine.WorkflowEvent:
						server.observer.OnWorkflowEvent(e)
					case engine.TaskEvent:
						server.observer.OnTaskEvent(e)
					}
					time.Sleep(10 * time.Millisecond)
				}
				// Wait for flush ticker (100ms) plus buffer
				time.Sleep(120 * time.Millisecond)
			}

			// Wait for completion or timeout
			err := <-errChan

			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			} else {
				// Context cancellation or EOF is expected
				if err != nil {
					st, ok := status.FromError(err)
					if ok {
						assert.Contains(t, []codes.Code{codes.Canceled, codes.Internal}, st.Code())
					}
				}
				assert.GreaterOrEqual(t, len(stream.responses), tt.wantResponses)
			}
		})
	}
}

func TestConvertWorkflowEventTypeToStatus(t *testing.T) {
	tests := []struct {
		eventType engine.WorkflowEventType
		want      pb.WorkflowStatus
	}{
		{engine.WorkflowEventSubmitted, pb.WorkflowStatus_WORKFLOW_STATUS_PENDING},
		{engine.WorkflowEventStarted, pb.WorkflowStatus_WORKFLOW_STATUS_RUNNING},
		{engine.WorkflowEventCompleted, pb.WorkflowStatus_WORKFLOW_STATUS_COMPLETED},
		{engine.WorkflowEventFailed, pb.WorkflowStatus_WORKFLOW_STATUS_FAILED},
		{engine.WorkflowEventCancelled, pb.WorkflowStatus_WORKFLOW_STATUS_CANCELLED},
	}

	for _, tt := range tests {
		t.Run(tt.want.String(), func(t *testing.T) {
			got := convertWorkflowEventTypeToStatus(tt.eventType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertTaskEventTypeToStatus(t *testing.T) {
	tests := []struct {
		eventType engine.TaskEventType
		want      pb.TaskStatus
	}{
		{engine.TaskEventStarted, pb.TaskStatus_TASK_STATUS_RUNNING},
		{engine.TaskEventProgress, pb.TaskStatus_TASK_STATUS_RUNNING},
		{engine.TaskEventCompleted, pb.TaskStatus_TASK_STATUS_COMPLETED},
		{engine.TaskEventFailed, pb.TaskStatus_TASK_STATUS_FAILED},
		{engine.TaskEventCancelled, pb.TaskStatus_TASK_STATUS_CANCELLED},
	}

	for _, tt := range tests {
		t.Run(tt.want.String(), func(t *testing.T) {
			got := convertTaskEventTypeToStatus(tt.eventType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsTerminalTaskEvent(t *testing.T) {
	tests := []struct {
		eventType engine.TaskEventType
		want      bool
	}{
		{engine.TaskEventStarted, false},
		{engine.TaskEventProgress, false},
		{engine.TaskEventCompleted, true},
		{engine.TaskEventFailed, true},
		{engine.TaskEventCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.eventType.String(), func(t *testing.T) {
			got := isTerminalTaskEvent(tt.eventType)
			assert.Equal(t, tt.want, got)
		})
	}
}
