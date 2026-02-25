## ADDED Requirements

### Requirement: Steer message pattern

The system SHALL support a steer message pattern that allows runtime modification of task parameters.

#### Scenario: Send steer signal to running task
- **WHEN** a steer signal is sent to a running task with new parameters
- **THEN** the task receives the signal and can update its behavior accordingly

#### Scenario: Steer signal to non-existent task
- **WHEN** a steer signal is sent to a task that does not exist
- **THEN** the system returns a task-not-found error

#### Scenario: Steer signal to completed task
- **WHEN** a steer signal is sent to a task that has already completed
- **THEN** the system returns a task-not-running error

#### Scenario: Multiple steer signals
- **WHEN** multiple steer signals are sent to the same task
- **THEN** the task receives all signals in order and applies the latest parameters

### Requirement: Interrupt message pattern

The system SHALL support an interrupt message pattern that stops a running task.

#### Scenario: Graceful interrupt
- **WHEN** an interrupt signal with graceful=true is sent to a running task
- **THEN** the task's context is cancelled and the task has time to clean up before termination

#### Scenario: Forced interrupt with timeout
- **WHEN** an interrupt signal with graceful=true and timeout=5s is sent
- **THEN** the task has 5 seconds to clean up before being forcefully terminated

#### Scenario: Immediate interrupt
- **WHEN** an interrupt signal with graceful=false is sent to a running task
- **THEN** the task's context is cancelled immediately

#### Scenario: Interrupt with reason
- **WHEN** an interrupt signal includes a reason field
- **THEN** the reason is recorded in the task's state and available for querying

#### Scenario: Interrupt non-running task
- **WHEN** an interrupt signal is sent to a pending task
- **THEN** the task is removed from the queue and marked as cancelled

### Requirement: Collect message pattern

The system SHALL support a collect message pattern that aggregates outputs from multiple tasks.

#### Scenario: Collect all task results
- **WHEN** a collector is created for tasks ["task-1", "task-2", "task-3"]
- **THEN** the collector waits for all tasks to complete and returns aggregated results

#### Scenario: Collect with timeout
- **WHEN** a collector is created with a 30s timeout and not all tasks complete in time
- **THEN** the collector returns partial results with a timeout error indicating which tasks are missing

#### Scenario: Collect with streaming
- **WHEN** a collector is created in streaming mode
- **THEN** the collector emits results as each task completes (fan-in pattern)

#### Scenario: Collect with all tasks failed
- **WHEN** all tasks in a collector fail
- **THEN** the collector returns an error with all individual task errors

#### Scenario: Collect with partial failure
- **WHEN** some tasks succeed and some fail
- **THEN** the collector returns successful results and errors for failed tasks

### Requirement: Signal context injection

The system SHALL inject a signal channel into task context for receiving runtime messages.

#### Scenario: Task receives signal from context
- **WHEN** a task accesses signals via `signal.FromContext(ctx)`
- **THEN** the task receives a channel that delivers signals sent to its task ID

#### Scenario: Task without signal subscription
- **WHEN** a task does not call `signal.FromContext(ctx)`
- **THEN** signals sent to the task are buffered (up to buffer limit) and discarded if not consumed

### Requirement: Signal payload validation

The system SHALL validate signal payloads before delivery.

#### Scenario: Valid steer payload
- **WHEN** a steer signal is sent with a map of key-value parameters
- **THEN** the signal is delivered to the task

#### Scenario: Invalid signal type
- **WHEN** a signal with an unknown type is sent
- **THEN** the system returns an invalid-signal-type error

#### Scenario: Empty task ID
- **WHEN** a signal is sent with an empty task ID
- **THEN** the system returns an invalid-task-id error

### Requirement: Message pattern metrics

The system SHALL expose metrics for message pattern operations.

#### Scenario: Track steer signals sent
- **WHEN** steer signals are sent
- **THEN** the system increments `signal_steer_total` metric

#### Scenario: Track interrupt signals sent
- **WHEN** interrupt signals are sent
- **THEN** the system increments `signal_interrupt_total` metric

#### Scenario: Track collect operations
- **WHEN** collect operations complete
- **THEN** the system records `signal_collect_duration_seconds` and `signal_collect_total` metrics

#### Scenario: Track signal delivery failures
- **WHEN** a signal fails to deliver (task not found, buffer full)
- **THEN** the system increments `signal_delivery_failures_total` metric

### Requirement: gRPC SignalTask integration

The system SHALL integrate message patterns with the gRPC `SignalTask` RPC.

#### Scenario: Send steer via gRPC
- **WHEN** `SignalTask` RPC is called with type=STEER
- **THEN** the system publishes a steer signal to the Signal Bus

#### Scenario: Send interrupt via gRPC
- **WHEN** `SignalTask` RPC is called with type=INTERRUPT
- **THEN** the system publishes an interrupt signal to the Signal Bus

#### Scenario: Send collect via gRPC
- **WHEN** `SignalTask` RPC is called with type=COLLECT and multiple task IDs
- **THEN** the system creates a collector and returns aggregated results
