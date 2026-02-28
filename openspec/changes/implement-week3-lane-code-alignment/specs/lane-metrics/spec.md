## ADDED Requirements

### Requirement: Backpressure outcome metrics
Lane metrics SHALL expose canonical backpressure outcomes using `accepted`, `rejected`, `redirected`, and `dropped` counters.

#### Scenario: Record accepted submissions
- **WHEN** a task submission is admitted into a lane queue
- **THEN** metrics MUST increment `accepted` for that lane

#### Scenario: Record rejected submissions
- **WHEN** a task submission fails before admission
- **THEN** metrics MUST increment `rejected` for that lane

#### Scenario: Record redirected submissions
- **WHEN** a task submission is redirected to another lane
- **THEN** metrics MUST increment `redirected` for source lane and MUST NOT classify the same event as direct accepted

#### Scenario: Record dropped submissions
- **WHEN** a task submission is dropped due to backpressure policy
- **THEN** metrics MUST increment `dropped` for that lane
