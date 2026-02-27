package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	dgbadger "github.com/dgraph-io/badger/v4"
	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"github.com/goclaw/goclaw/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type mockSagaWatchStream struct {
	ctx     context.Context
	events  []*pb.WatchSagaEvent
	sendErr error
}

func (m *mockSagaWatchStream) Send(event *pb.WatchSagaEvent) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.events = append(m.events, event)
	return nil
}

func (m *mockSagaWatchStream) Context() context.Context {
	return m.ctx
}

func (m *mockSagaWatchStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockSagaWatchStream) SendHeader(metadata.MD) error { return nil }
func (m *mockSagaWatchStream) SetTrailer(metadata.MD)       {}
func (m *mockSagaWatchStream) SendMsg(any) error            { return nil }
func (m *mockSagaWatchStream) RecvMsg(any) error            { return nil }

func newSagaServiceForTest(t *testing.T) (*SagaServiceServer, func()) {
	t.Helper()

	opts := dgbadger.DefaultOptions(t.TempDir())
	opts.Logger = nil
	db, err := dgbadger.Open(opts)
	require.NoError(t, err)

	wal, err := saga.NewBadgerWAL(db, saga.WALOptions{WriteMode: saga.WALWriteModeSync})
	require.NoError(t, err)

	checkpointStore, err := saga.NewBadgerCheckpointStore(db)
	require.NoError(t, err)

	checkpointer, err := saga.NewCheckpointer(checkpointStore)
	require.NoError(t, err)

	orchestrator := saga.NewSagaOrchestrator(
		saga.WithWAL(wal),
		saga.WithCheckpointer(checkpointer),
		saga.WithSagaStore(saga.NewMemorySagaStore()),
	)

	server := NewSagaServiceServer(orchestrator, checkpointStore)
	return server, func() {
		_ = wal.Close()
		_ = db.Close()
	}
}

func TestSagaServiceSubmitGetList(t *testing.T) {
	server, cleanup := newSagaServiceForTest(t)
	defer cleanup()

	input, err := structpb.NewStruct(map[string]any{"order_id": "o-123"})
	require.NoError(t, err)

	submitResp, err := server.SubmitSaga(context.Background(), &pb.SubmitSagaRequest{
		Name: "grpc-saga",
		Steps: []*pb.SagaStepDefinition{
			{Id: "a"},
		},
		Input: input,
	})
	require.NoError(t, err)
	require.NotEmpty(t, submitResp.SagaId)
	assert.Equal(t, pb.SagaState_SAGA_STATE_RUNNING, submitResp.State)

	time.Sleep(80 * time.Millisecond)

	getResp, err := server.GetSagaStatus(context.Background(), &pb.GetSagaStatusRequest{SagaId: submitResp.SagaId})
	require.NoError(t, err)
	assert.Equal(t, submitResp.SagaId, getResp.SagaId)
	assert.Contains(t, []pb.SagaState{pb.SagaState_SAGA_STATE_COMPLETED, pb.SagaState_SAGA_STATE_RUNNING}, getResp.State)

	listResp, err := server.ListSagas(context.Background(), &pb.ListSagasRequest{
		StateFilter: pb.SagaState_SAGA_STATE_COMPLETED,
		Pagination:  &pb.PaginationRequest{PageSize: 10},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Sagas), 1)
}

func TestSagaServiceCompensate(t *testing.T) {
	server, cleanup := newSagaServiceForTest(t)
	defer cleanup()

	submitResp, err := server.SubmitSaga(context.Background(), &pb.SubmitSagaRequest{
		Name:   "manual-compensate",
		Policy: pb.SagaCompensationPolicy_SAGA_COMPENSATION_POLICY_MANUAL,
		Steps: []*pb.SagaStepDefinition{
			{Id: "a", EnableCompensation: true},
			{Id: "b", DependsOn: []string{"a"}, ShouldFail: true},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, submitResp.SagaId)

	time.Sleep(100 * time.Millisecond)

	compResp, err := server.CompensateSaga(context.Background(), &pb.CompensateSagaRequest{
		SagaId: submitResp.SagaId,
		Reason: "manual",
	})
	require.NoError(t, err)
	assert.Equal(t, submitResp.SagaId, compResp.SagaId)
	assert.Equal(t, pb.SagaState_SAGA_STATE_COMPENSATED, compResp.State)

	_, err = server.CompensateSaga(context.Background(), &pb.CompensateSagaRequest{
		SagaId: submitResp.SagaId,
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestSagaServiceWatchSaga(t *testing.T) {
	server, cleanup := newSagaServiceForTest(t)
	defer cleanup()

	submitResp, err := server.SubmitSaga(context.Background(), &pb.SubmitSagaRequest{
		Name: "watch-saga",
		Steps: []*pb.SagaStepDefinition{
			{Id: "a", DelayMs: 20},
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	stream := &mockSagaWatchStream{
		ctx:    ctx,
		events: make([]*pb.WatchSagaEvent, 0),
	}

	err = server.WatchSaga(&pb.WatchSagaRequest{
		SagaId:         submitResp.SagaId,
		PollIntervalMs: 20,
	}, stream)
	require.NoError(t, err)
	require.NotEmpty(t, stream.events)
	assert.Equal(t, submitResp.SagaId, stream.events[len(stream.events)-1].SagaId)
	assert.Contains(
		t,
		[]pb.SagaState{
			pb.SagaState_SAGA_STATE_COMPLETED,
			pb.SagaState_SAGA_STATE_COMPENSATED,
			pb.SagaState_SAGA_STATE_COMPENSATION_FAILED,
		},
		stream.events[len(stream.events)-1].State,
	)
}

func TestSagaServiceErrorCodes(t *testing.T) {
	server, cleanup := newSagaServiceForTest(t)
	defer cleanup()

	_, err := server.SubmitSaga(context.Background(), &pb.SubmitSagaRequest{
		Name:  "invalid",
		Steps: []*pb.SagaStepDefinition{{Id: "a", DependsOn: []string{"missing"}}},
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())

	_, err = server.GetSagaStatus(context.Background(), &pb.GetSagaStatusRequest{SagaId: "missing"})
	require.Error(t, err)
	st, ok = status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())

	_, err = server.CompensateSaga(context.Background(), &pb.CompensateSagaRequest{SagaId: "missing"})
	require.Error(t, err)
	st, ok = status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestBuildSagaDefinitionFromProto(t *testing.T) {
	definition, input, err := buildSagaDefinitionFromProto(&pb.SubmitSagaRequest{
		Name:   "convert",
		Policy: pb.SagaCompensationPolicy_SAGA_COMPENSATION_POLICY_SKIP,
		Steps: []*pb.SagaStepDefinition{
			{Id: "a"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, definition)
	assert.Equal(t, saga.SkipCompensate, definition.Policy)
	assert.IsType(t, map[string]any{}, input)
}

func TestProtoStateFilterToString(t *testing.T) {
	state, err := protoStateFilterToString(pb.SagaState_SAGA_STATE_COMPENSATING)
	require.NoError(t, err)
	assert.Equal(t, "compensating", state)

	_, err = protoStateFilterToString(pb.SagaState(999))
	require.Error(t, err)
}

func TestParseOffsetToken(t *testing.T) {
	offset, err := parseOffsetToken("15")
	require.NoError(t, err)
	assert.Equal(t, 15, offset)

	_, err = parseOffsetToken("-1")
	require.Error(t, err)

	_, err = parseOffsetToken("abc")
	require.Error(t, err)
}

func TestSagaInstanceToStatusMarshalError(t *testing.T) {
	instance := &saga.SagaInstance{
		ID:             "s-1",
		DefinitionName: "bad-result",
		State:          saga.SagaStateRunning,
		StepResults: map[string]any{
			"a": func() {},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, err := sagaInstanceToStatus(instance)
	require.Error(t, err)
	var marshalErr *json.UnsupportedTypeError
	assert.True(t, errors.As(err, &marshalErr))
}
