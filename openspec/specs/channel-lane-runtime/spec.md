# channel-lane-runtime Specification

## Purpose
Define canonical in-memory channel-lane runtime semantics for worker concurrency, admission control, ordering, lifecycle safety, and backpressure outcome accounting.

## Requirements

### Requirement: Configurable worker concurrency model
Channel lane runtime SHALL support configurable worker concurrency with fixed-by-default behavior and optional dynamic scaling support.

#### Scenario: Fixed concurrency by default
- **WHEN** a lane is created with `MaxConcurrency = N` and dynamic scaling is not enabled
- **THEN** runtime executes tasks with at most `N` concurrent workers

#### Scenario: Optional dynamic scaling path
- **WHEN** dynamic scaling is enabled by runtime configuration
- **THEN** runtime MAY adjust active worker count within configured bounds without changing submission API

### Requirement: Token bucket admission baseline
Channel lane admission control SHALL treat Token Bucket as the normative rate-limiting baseline for Week3 runtime semantics.

#### Scenario: Token bucket allows immediate admission
- **WHEN** token budget is available
- **THEN** task admission proceeds without additional wait

#### Scenario: Token bucket limits admission
- **WHEN** token budget is exhausted
- **THEN** admission follows configured wait/reject behavior and does not bypass token checks

### Requirement: Deterministic tie-breaking for equal-priority tasks
Priority-enabled channel lane SHALL process equal-priority tasks in deterministic order.

#### Scenario: Equal-priority FIFO tie-break
- **WHEN** two tasks with equal priority are enqueued in order A then B
- **THEN** dequeue order MUST be A then B

### Requirement: Idempotent lifecycle close semantics
Channel lane close lifecycle SHALL be safe for repeated calls.

#### Scenario: Repeated close invocation
- **WHEN** `Close` is called more than once on the same lane
- **THEN** runtime MUST not panic and lane closed state remains consistent

### Requirement: Concurrent-safe manager interactions
Lane manager interactions used by channel lane runtime SHALL be safe under concurrent read/write access.

#### Scenario: Concurrent lookup and submission
- **WHEN** goroutines concurrently perform lane registration/lookup/submission
- **THEN** manager state remains valid and operations complete without data race or structural corruption

### Requirement: Canonical backpressure outcome accounting
Channel lane submission paths SHALL account for backpressure outcomes using canonical categories: `accepted`, `rejected`, `redirected`, and `dropped`.

#### Scenario: Drop strategy accounting
- **WHEN** a submission is dropped due to full queue in Drop mode
- **THEN** `dropped` MUST increment and `accepted` MUST NOT increment for that submission

#### Scenario: Redirect strategy accounting
- **WHEN** a full-queue submission is successfully redirected
- **THEN** `redirected` MUST increment and outcome classification MUST remain distinct from direct acceptance

#### Scenario: Redirect target failure accounting
- **WHEN** redirect path is attempted but target lane submission fails
- **THEN** source lane MUST NOT increment `redirected` for that submission
- **AND** source lane MUST classify the terminal outcome as non-redirect success (`dropped` or `rejected` according to path semantics)

#### Scenario: Rejected submission accounting
- **WHEN** a submission fails before admission (for example due to context cancellation)
- **THEN** `rejected` MUST increment and task MUST not be counted as accepted
