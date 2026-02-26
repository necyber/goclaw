package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

// RedisBus is a Redis Pub/Sub-backed Signal Bus implementation.
type RedisBus struct {
	client        redis.UniversalClient
	channelPrefix string
	bufferSize    int

	mu          sync.RWMutex
	subscribers map[string]*redisSubscription
	closed      bool
}

type redisSubscription struct {
	pubsub *redis.PubSub
	ch     chan *Signal
	cancel context.CancelFunc
}

// NewRedisBus creates a new Redis-backed Signal Bus.
func NewRedisBus(client redis.UniversalClient, channelPrefix string, bufferSize int) *RedisBus {
	if channelPrefix == "" {
		channelPrefix = "goclaw:signal:"
	}
	if bufferSize <= 0 {
		bufferSize = 16
	}
	return &RedisBus{
		client:        client,
		channelPrefix: channelPrefix,
		bufferSize:    bufferSize,
		subscribers:   make(map[string]*redisSubscription),
	}
}

// Publish sends a signal via Redis Pub/Sub.
func (b *RedisBus) Publish(ctx context.Context, sig *Signal) error {
	if sig == nil {
		metricsRecorder().RecordSignalFailed("redis", "unknown", "nil_signal")
		return fmt.Errorf("signal cannot be nil")
	}
	if sig.TaskID == "" {
		metricsRecorder().RecordSignalFailed("redis", string(sig.Type), "empty_task_id")
		return fmt.Errorf("signal task_id cannot be empty")
	}

	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		metricsRecorder().RecordSignalFailed("redis", string(sig.Type), "bus_closed")
		return fmt.Errorf("signal bus is closed")
	}
	b.mu.RUnlock()

	data, err := json.Marshal(sig)
	if err != nil {
		metricsRecorder().RecordSignalFailed("redis", string(sig.Type), "marshal_failed")
		return fmt.Errorf("failed to marshal signal: %w", err)
	}

	channel := b.channelPrefix + sig.TaskID
	if err := b.client.Publish(ctx, channel, data).Err(); err != nil {
		metricsRecorder().RecordSignalFailed("redis", string(sig.Type), "publish_failed")
		return err
	}
	metricsRecorder().RecordSignalSent("redis", string(sig.Type))
	return nil
}

// Subscribe creates a channel that receives signals for the given task via Redis Pub/Sub.
func (b *RedisBus) Subscribe(ctx context.Context, taskID string) (<-chan *Signal, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id cannot be empty")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, fmt.Errorf("signal bus is closed")
	}

	if _, exists := b.subscribers[taskID]; exists {
		return nil, fmt.Errorf("task %s already subscribed", taskID)
	}

	channel := b.channelPrefix + taskID
	pubsub := b.client.Subscribe(ctx, channel)

	ch := make(chan *Signal, b.bufferSize)
	subCtx, cancel := context.WithCancel(ctx)

	sub := &redisSubscription{
		pubsub: pubsub,
		ch:     ch,
		cancel: cancel,
	}
	b.subscribers[taskID] = sub

	// Background goroutine to forward Redis messages to the channel.
	go b.forwardMessages(subCtx, pubsub, ch)

	return ch, nil
}

func (b *RedisBus) forwardMessages(ctx context.Context, pubsub *redis.PubSub, ch chan *Signal) {
	defer func() {
		_ = pubsub.Close()
	}()

	redisCh := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-redisCh:
			if !ok {
				return
			}
			var sig Signal
			if err := json.Unmarshal([]byte(msg.Payload), &sig); err != nil {
				metricsRecorder().RecordSignalFailed("redis", "unknown", "decode_failed")
				continue
			}
			select {
			case ch <- &sig:
				metricsRecorder().RecordSignalReceived("redis", string(sig.Type))
			default:
				metricsRecorder().RecordSignalFailed("redis", string(sig.Type), "buffer_full_drop")
				select {
				case <-ch:
				default:
				}
				select {
				case ch <- &sig:
					metricsRecorder().RecordSignalReceived("redis", string(sig.Type))
				default:
					metricsRecorder().RecordSignalFailed("redis", string(sig.Type), "buffer_still_full")
				}
			}
		}
	}
}

// Unsubscribe removes the Redis subscription for the given task.
func (b *RedisBus) Unsubscribe(taskID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, ok := b.subscribers[taskID]
	if !ok {
		return nil
	}

	sub.cancel()
	close(sub.ch)
	delete(b.subscribers, taskID)
	return nil
}

// Close shuts down all subscriptions and the bus.
func (b *RedisBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	for taskID, sub := range b.subscribers {
		sub.cancel()
		close(sub.ch)
		delete(b.subscribers, taskID)
	}
	return nil
}

// Healthy checks if the Redis connection is alive.
func (b *RedisBus) Healthy() bool {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return false
	}
	b.mu.RUnlock()

	return b.client.Ping(context.Background()).Err() == nil
}
