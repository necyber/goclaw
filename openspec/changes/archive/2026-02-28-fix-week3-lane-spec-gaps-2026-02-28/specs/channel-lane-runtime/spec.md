## MODIFIED Requirements

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
