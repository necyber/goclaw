package eventbus

import "fmt"

const (
	// SubjectPrefix is the canonical prefix for distributed lifecycle events.
	SubjectPrefix = "goclaw.v1.lifecycle"
)

// Domain identifies workflow/task lifecycle event domains.
type Domain string

const (
	DomainWorkflow Domain = "workflow"
	DomainTask     Domain = "task"
)

// WorkflowSubject returns canonical workflow lifecycle subject.
func WorkflowSubject(shardKey, eventType string) string {
	return fmt.Sprintf("%s.%s.%s.%s", SubjectPrefix, DomainWorkflow, sanitizeSegment(shardKey), sanitizeSegment(eventType))
}

// TaskSubject returns canonical task lifecycle subject.
func TaskSubject(shardKey, eventType string) string {
	return fmt.Sprintf("%s.%s.%s.%s", SubjectPrefix, DomainTask, sanitizeSegment(shardKey), sanitizeSegment(eventType))
}

// DomainWildcardSubject returns canonical wildcard subject for a domain.
func DomainWildcardSubject(domain Domain) string {
	return fmt.Sprintf("%s.%s.>", SubjectPrefix, sanitizeSegment(string(domain)))
}

func sanitizeSegment(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
