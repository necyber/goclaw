# Rate Limiter Spec (Week 3 Archive Backfill)

## Scope

Define lane-level rate limiting behavior for task admission control.

## Requirements

### FR-1 Token bucket support

Rate limiter SHALL support token bucket mode with configurable refill rate and burst capacity.

### FR-2 Wait/allow APIs

Rate limiter SHALL provide:

- Immediate allow/reject check.
- Optional wait path for admission.

### FR-3 Integration with submission path

Lane submission SHALL enforce rate limit checks before queue insertion.

### FR-4 Predictable behavior under load

Rate limiting SHALL remain stable under burst traffic without unbounded memory growth.

## Archive Note

Historical backfill for archived change `week3-lane-queue-system`.

