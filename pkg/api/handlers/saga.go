package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/api/response"
	"github.com/goclaw/goclaw/pkg/logger"
	"github.com/goclaw/goclaw/pkg/saga"
	"github.com/google/uuid"
)

// SagaHandler handles Saga API endpoints.
type SagaHandler struct {
	orchestrator    *saga.SagaOrchestrator
	checkpointStore saga.CheckpointStore
	recoveryManager *saga.RecoveryManager
	logger          logger.Logger
	validator       *validator.Validate

	defMu       sync.RWMutex
	definitions map[string]*saga.SagaDefinition
}

// NewSagaHandler creates a Saga handler.
func NewSagaHandler(
	orchestrator *saga.SagaOrchestrator,
	checkpointStore saga.CheckpointStore,
	recoveryManager *saga.RecoveryManager,
	log logger.Logger,
) *SagaHandler {
	return &SagaHandler{
		orchestrator:    orchestrator,
		checkpointStore: checkpointStore,
		recoveryManager: recoveryManager,
		logger:          log,
		validator:       validator.New(),
		definitions:     make(map[string]*saga.SagaDefinition),
	}
}

// SubmitSaga handles POST /api/v1/sagas.
// @Summary Submit a saga
// @Description Submit a saga definition for asynchronous execution
// @Tags sagas
// @Accept json
// @Produce json
// @Param saga body models.SagaSubmitRequest true "Saga submit request"
// @Success 201 {object} models.SagaSubmitResponse "Saga accepted"
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 503 {object} response.ErrorResponse "Saga runtime unavailable"
// @Router /api/v1/sagas [post]
func (h *SagaHandler) SubmitSaga(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		response.Error(w, http.StatusServiceUnavailable, response.ErrCodeServiceUnavailable, "saga orchestrator unavailable", getRequestID(r.Context()))
		return
	}

	var req models.SagaSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "invalid request body", getRequestID(r.Context()))
		return
	}
	if err := h.validator.Struct(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.ErrCodeValidationFailed, err.Error(), getRequestID(r.Context()))
		return
	}

	definition, err := buildSagaDefinition(req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.ErrCodeValidationFailed, err.Error(), getRequestID(r.Context()))
		return
	}

	sagaID := uuid.NewString()
	h.defMu.Lock()
	h.definitions[sagaID] = definition
	h.defMu.Unlock()

	input := any(req.Input)
	go func() {
		_, execErr := h.orchestrator.ExecuteWithID(context.Background(), sagaID, definition, input)
		if execErr != nil && h.logger != nil {
			h.logger.Warn("saga execution finished with error", "saga_id", sagaID, "error", execErr)
		}
	}()

	resp := models.SagaSubmitResponse{
		SagaID:    sagaID,
		Name:      definition.Name,
		Status:    saga.SagaStateRunning.String(),
		CreatedAt: time.Now().UTC(),
	}
	response.JSON(w, http.StatusCreated, resp)
}

// GetSaga handles GET /api/v1/sagas/{id}.
// @Summary Get saga status
// @Description Get runtime status for a saga instance
// @Tags sagas
// @Produce json
// @Param id path string true "Saga ID"
// @Success 200 {object} models.SagaStatusResponse "Saga status"
// @Failure 400 {object} response.ErrorResponse "Invalid saga ID"
// @Failure 404 {object} response.ErrorResponse "Saga not found"
// @Failure 503 {object} response.ErrorResponse "Saga runtime unavailable"
// @Router /api/v1/sagas/{id} [get]
func (h *SagaHandler) GetSaga(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		response.Error(w, http.StatusServiceUnavailable, response.ErrCodeServiceUnavailable, "saga orchestrator unavailable", getRequestID(r.Context()))
		return
	}

	sagaID := chi.URLParam(r, "id")
	if sagaID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "saga id is required", getRequestID(r.Context()))
		return
	}

	instance, err := h.orchestrator.GetInstance(sagaID)
	if err != nil {
		response.Error(w, http.StatusNotFound, response.ErrCodeNotFound, "saga not found", getRequestID(r.Context()))
		return
	}

	resp := models.SagaStatusResponse{
		SagaID:         instance.ID,
		Name:           instance.DefinitionName,
		State:          instance.State.String(),
		CompletedSteps: append([]string(nil), instance.CompletedSteps...),
		Compensated:    append([]string(nil), instance.Compensated...),
		FailedStep:     instance.FailedStep,
		FailureReason:  instance.FailureReason,
		StepResults:    sagaResultMap(instance.StepResults),
		CreatedAt:      instance.CreatedAt,
		UpdatedAt:      instance.UpdatedAt,
		StartedAt:      instance.StartedAt,
		CompletedAt:    instance.CompletedAt,
	}
	response.JSON(w, http.StatusOK, resp)
}

// ListSagas handles GET /api/v1/sagas.
// @Summary List sagas
// @Description List saga instances with optional state filter and pagination
// @Tags sagas
// @Produce json
// @Param state query string false "Filter by saga state"
// @Param limit query int false "Maximum number of results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} models.SagaListResponse "Saga list"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Failure 503 {object} response.ErrorResponse "Saga runtime unavailable"
// @Router /api/v1/sagas [get]
func (h *SagaHandler) ListSagas(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		response.Error(w, http.StatusServiceUnavailable, response.ErrCodeServiceUnavailable, "saga orchestrator unavailable", getRequestID(r.Context()))
		return
	}

	limit := 20
	offset := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			limit = parsed
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))

	instances, total, err := h.orchestrator.ListInstancesFiltered(r.Context(), saga.SagaListFilter{
		State:  state,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, err.Error(), getRequestID(r.Context()))
		return
	}

	items := make([]models.SagaSummary, 0, len(instances))
	for _, instance := range instances {
		items = append(items, models.SagaSummary{
			SagaID:      instance.ID,
			Name:        instance.DefinitionName,
			State:       instance.State.String(),
			CreatedAt:   instance.CreatedAt,
			CompletedAt: instance.CompletedAt,
		})
	}

	response.JSON(w, http.StatusOK, models.SagaListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// CompensateSaga handles POST /api/v1/sagas/{id}/compensate.
// @Summary Trigger saga compensation
// @Description Manually trigger compensation for a saga in pending-compensation state
// @Tags sagas
// @Accept json
// @Produce json
// @Param id path string true "Saga ID"
// @Param request body models.SagaCompensateRequest false "Compensation reason"
// @Success 202 {object} models.SagaActionResponse "Compensation started"
// @Failure 400 {object} response.ErrorResponse "Invalid saga ID"
// @Failure 404 {object} response.ErrorResponse "Saga not found"
// @Failure 409 {object} response.ErrorResponse "Invalid saga state"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Failure 503 {object} response.ErrorResponse "Saga runtime unavailable"
// @Router /api/v1/sagas/{id}/compensate [post]
func (h *SagaHandler) CompensateSaga(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		response.Error(w, http.StatusServiceUnavailable, response.ErrCodeServiceUnavailable, "saga orchestrator unavailable", getRequestID(r.Context()))
		return
	}

	sagaID := chi.URLParam(r, "id")
	if sagaID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "saga id is required", getRequestID(r.Context()))
		return
	}

	definition := h.getDefinition(sagaID)
	if definition == nil {
		response.Error(w, http.StatusNotFound, response.ErrCodeNotFound, "saga definition not found", getRequestID(r.Context()))
		return
	}

	var req models.SagaCompensateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	reason := errors.New("manual compensation requested")
	if strings.TrimSpace(req.Reason) != "" {
		reason = errors.New(req.Reason)
	}

	instance, err := h.orchestrator.TriggerCompensation(r.Context(), sagaID, definition, nil, reason)
	if err != nil {
		if errors.Is(err, saga.ErrSagaNotFound) {
			response.Error(w, http.StatusNotFound, response.ErrCodeNotFound, "saga not found", getRequestID(r.Context()))
			return
		}
		if strings.Contains(err.Error(), "pending-compensation") {
			response.Error(w, http.StatusConflict, response.ErrCodeConflict, err.Error(), getRequestID(r.Context()))
			return
		}
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, err.Error(), getRequestID(r.Context()))
		return
	}

	response.JSON(w, http.StatusAccepted, models.SagaActionResponse{
		SagaID: sagaID,
		State:  instance.State.String(),
	})
}

// RecoverSaga handles POST /api/v1/sagas/{id}/recover.
// @Summary Recover saga from checkpoint
// @Description Resume a non-terminal saga from its latest checkpoint
// @Tags sagas
// @Produce json
// @Param id path string true "Saga ID"
// @Success 202 {object} models.SagaActionResponse "Recovery executed"
// @Failure 400 {object} response.ErrorResponse "Invalid saga ID"
// @Failure 404 {object} response.ErrorResponse "Definition or checkpoint not found"
// @Failure 409 {object} response.ErrorResponse "Saga already terminal"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Failure 503 {object} response.ErrorResponse "Saga recovery unavailable"
// @Router /api/v1/sagas/{id}/recover [post]
func (h *SagaHandler) RecoverSaga(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil || h.checkpointStore == nil {
		response.Error(w, http.StatusServiceUnavailable, response.ErrCodeServiceUnavailable, "saga recovery unavailable", getRequestID(r.Context()))
		return
	}

	sagaID := chi.URLParam(r, "id")
	if sagaID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "saga id is required", getRequestID(r.Context()))
		return
	}
	definition := h.getDefinition(sagaID)
	if definition == nil {
		response.Error(w, http.StatusNotFound, response.ErrCodeNotFound, "saga definition not found", getRequestID(r.Context()))
		return
	}

	checkpoint, err := h.checkpointStore.Load(r.Context(), sagaID)
	if err != nil {
		response.Error(w, http.StatusNotFound, response.ErrCodeNotFound, "checkpoint not found", getRequestID(r.Context()))
		return
	}
	if checkpoint.State.IsTerminal() {
		response.Error(w, http.StatusConflict, response.ErrCodeConflict, "saga is already in terminal state", getRequestID(r.Context()))
		return
	}

	instance, err := h.orchestrator.ResumeFromCheckpoint(r.Context(), definition, checkpoint, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, err.Error(), getRequestID(r.Context()))
		return
	}

	response.JSON(w, http.StatusAccepted, models.SagaActionResponse{
		SagaID: sagaID,
		State:  instance.State.String(),
	})
}

func (h *SagaHandler) getDefinition(sagaID string) *saga.SagaDefinition {
	h.defMu.RLock()
	defer h.defMu.RUnlock()
	return h.definitions[sagaID]
}

func buildSagaDefinition(req models.SagaSubmitRequest) (*saga.SagaDefinition, error) {
	builder := saga.New(req.Name)
	if req.TimeoutMS > 0 {
		builder = builder.WithTimeout(time.Duration(req.TimeoutMS) * time.Millisecond)
	}
	if req.StepTimeoutMS > 0 {
		builder = builder.WithDefaultStepTimeout(time.Duration(req.StepTimeoutMS) * time.Millisecond)
	}
	switch strings.ToLower(strings.TrimSpace(req.Policy)) {
	case "", "auto":
		builder = builder.WithCompensationPolicy(saga.AutoCompensate)
	case "manual":
		builder = builder.WithCompensationPolicy(saga.ManualCompensate)
	case "skip":
		builder = builder.WithCompensationPolicy(saga.SkipCompensate)
	default:
		return nil, fmt.Errorf("unsupported policy: %s", req.Policy)
	}

	for _, stepReq := range req.Steps {
		stepReq := stepReq
		options := []saga.StepOption{
			saga.Action(func(ctx context.Context, stepCtx *saga.StepContext) (any, error) {
				if stepReq.DelayMS > 0 {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(time.Duration(stepReq.DelayMS) * time.Millisecond):
					}
				}
				if stepReq.ShouldFail {
					return nil, fmt.Errorf("step %s failed by request", stepReq.ID)
				}
				return map[string]any{
					"step_id": stepReq.ID,
					"saga_id": stepCtx.SagaID,
					"status":  "ok",
				}, nil
			}),
		}
		if len(stepReq.DependsOn) > 0 {
			options = append(options, saga.DependsOn(stepReq.DependsOn...))
		}
		if stepReq.TimeoutMS > 0 {
			options = append(options, saga.StepTimeout(time.Duration(stepReq.TimeoutMS)*time.Millisecond))
		}
		if stepReq.SkipCompensation {
			options = append(options, saga.WithStepCompensationPolicy(saga.SkipCompensate))
		}
		if stepReq.EnableCompensation {
			options = append(options, saga.Compensate(func(ctx context.Context, c *saga.CompensationContext) error {
				if stepReq.DelayMS > 0 {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(time.Duration(stepReq.DelayMS) * time.Millisecond):
					}
				}
				return nil
			}))
		}

		builder = builder.Step(stepReq.ID, options...)
	}
	return builder.Build()
}

func sagaResultMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	copied := make(map[string]any, len(source))
	for k, v := range source {
		copied[k] = v
	}
	return copied
}
