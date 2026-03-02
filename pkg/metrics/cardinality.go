package metrics

import (
	"regexp"
	"strings"
	"sync"
)

const defaultLabelCardinalityLimit = 1000

var nonLabelValueChars = regexp.MustCompile(`[^a-zA-Z0-9:_./-]`)

type labelCardinalityGuard struct {
	limit int
	logf  func(msg string, args ...any)

	mu         sync.Mutex
	values     map[string]map[string]struct{}
	warnedOnce map[string]bool
}

func newLabelCardinalityGuard(limit int, logf func(msg string, args ...any)) *labelCardinalityGuard {
	if limit <= 0 {
		limit = defaultLabelCardinalityLimit
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &labelCardinalityGuard{
		limit:      limit,
		logf:       logf,
		values:     make(map[string]map[string]struct{}),
		warnedOnce: make(map[string]bool),
	}
}

func sanitizeLabelValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	value = nonLabelValueChars.ReplaceAllString(value, "_")
	if value == "" {
		return "unknown"
	}
	return value
}

func (g *labelCardinalityGuard) admit(metricName, labelName, value string) (string, bool) {
	sanitized := sanitizeLabelValue(value)
	key := metricName + "|" + labelName

	g.mu.Lock()
	defer g.mu.Unlock()

	set, ok := g.values[key]
	if !ok {
		set = make(map[string]struct{})
		g.values[key] = set
	}
	if _, exists := set[sanitized]; exists {
		return sanitized, true
	}
	if len(set) >= g.limit {
		if !g.warnedOnce[key] {
			g.warnedOnce[key] = true
			g.logf("limit reached for %s label %s (limit=%d), dropping new label values", metricName, labelName, g.limit)
		}
		return "", false
	}
	set[sanitized] = struct{}{}
	return sanitized, true
}
