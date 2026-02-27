package eventbus

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Message is a delivered event-bus message.
type Message struct {
	Subject   string
	Payload   []byte
	Timestamp time.Time
}

// Subscription represents a stream subscription.
type Subscription struct {
	pattern string
	ch      chan Message
	bus     *MemoryBus
	once    sync.Once
}

// C returns read-only message channel.
func (s *Subscription) C() <-chan Message {
	return s.ch
}

// Close removes the subscription and closes its channel.
func (s *Subscription) Close() error {
	s.once.Do(func() {
		s.bus.unsubscribe(s.pattern, s.ch)
		close(s.ch)
	})
	return nil
}

// MemoryBus is an in-memory pub/sub transport useful for tests and local bridging.
type MemoryBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Message
}

// NewMemoryBus creates an in-memory event bus.
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		subscribers: make(map[string][]chan Message),
	}
}

// Publish publishes to all matching subscriptions.
func (b *MemoryBus) Publish(ctx context.Context, subject string, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if subject == "" {
		return fmt.Errorf("eventbus: subject cannot be empty")
	}

	b.mu.RLock()
	targets := make([]chan Message, 0)
	for pattern, channels := range b.subscribers {
		if !subjectMatches(pattern, subject) {
			continue
		}
		targets = append(targets, channels...)
	}
	b.mu.RUnlock()

	msg := Message{
		Subject:   subject,
		Payload:   append([]byte(nil), payload...),
		Timestamp: time.Now().UTC(),
	}
	for _, ch := range targets {
		select {
		case ch <- msg:
		default:
			// non-blocking drop for slow subscribers
		}
	}
	return nil
}

// Subscribe subscribes by subject pattern.
func (b *MemoryBus) Subscribe(pattern string, buffer int) (*Subscription, error) {
	if pattern == "" {
		return nil, fmt.Errorf("eventbus: subscription pattern cannot be empty")
	}
	if buffer <= 0 {
		buffer = 32
	}
	ch := make(chan Message, buffer)

	b.mu.Lock()
	b.subscribers[pattern] = append(b.subscribers[pattern], ch)
	b.mu.Unlock()

	return &Subscription{
		pattern: pattern,
		ch:      ch,
		bus:     b,
	}, nil
}

func (b *MemoryBus) unsubscribe(pattern string, target chan Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels := b.subscribers[pattern]
	filtered := channels[:0]
	for _, ch := range channels {
		if ch == target {
			continue
		}
		filtered = append(filtered, ch)
	}
	if len(filtered) == 0 {
		delete(b.subscribers, pattern)
		return
	}
	b.subscribers[pattern] = filtered
}

// subjectMatches supports exact, "*" segment, and ">" suffix wildcards.
func subjectMatches(pattern, subject string) bool {
	if pattern == subject {
		return true
	}
	if strings.HasSuffix(pattern, ".>") {
		prefix := strings.TrimSuffix(pattern, ".>")
		if prefix == "" {
			return true
		}
		return subject == prefix || strings.HasPrefix(subject, prefix+".")
	}

	patternParts := strings.Split(pattern, ".")
	subjectParts := strings.Split(subject, ".")
	if len(patternParts) != len(subjectParts) {
		return false
	}
	for i := range patternParts {
		if patternParts[i] == "*" {
			continue
		}
		if patternParts[i] != subjectParts[i] {
			return false
		}
	}
	return true
}
