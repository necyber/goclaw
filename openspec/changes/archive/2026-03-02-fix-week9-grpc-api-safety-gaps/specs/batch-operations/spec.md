## MODIFIED Requirements

### Requirement: Batch operation pagination
The system SHALL support pagination for large batch responses.

#### Scenario: Paginated batch results
- **WHEN** batch response exceeds page size
- **THEN** server MUST return first page with continuation token

#### Scenario: Continuation token
- **WHEN** client requests next page
- **THEN** server MUST return next batch of results using continuation token

#### Scenario: Page size configuration
- **WHEN** client specifies page size
- **THEN** server MUST respect page size up to maximum limit

#### Scenario: Invalid continuation token
- **WHEN** client provides a non-numeric or negative continuation token
- **THEN** server MUST return `InvalidArgument` with token validation details

#### Scenario: Continuation token beyond range
- **WHEN** client provides a continuation token offset greater than available input items
- **THEN** server MUST return `InvalidArgument` and MUST NOT panic
