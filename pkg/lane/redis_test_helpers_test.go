package lane

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

var errMockRedisUnavailable = errors.New("mock redis unavailable")

type mockZMember struct {
	member string
	score  float64
}

type mockRedisClient struct {
	redis.Cmdable

	mu    sync.Mutex
	lists map[string][]string
	zsets map[string][]mockZMember
	sets  map[string]map[string]struct{}
	down  atomic.Bool
}

func newMockRedisClient(t *testing.T) *mockRedisClient {
	t.Helper()

	return &mockRedisClient{
		lists: make(map[string][]string),
		zsets: make(map[string][]mockZMember),
		sets:  make(map[string]map[string]struct{}),
	}
}

func (m *mockRedisClient) SetDown(down bool) {
	m.down.Store(down)
}

func (m *mockRedisClient) Ping(_ context.Context) *redis.StatusCmd {
	if m.down.Load() {
		return redis.NewStatusResult("", errMockRedisUnavailable)
	}
	return redis.NewStatusResult("PONG", nil)
}

func (m *mockRedisClient) LLen(_ context.Context, key string) *redis.IntCmd {
	if m.down.Load() {
		return redis.NewIntResult(0, errMockRedisUnavailable)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return redis.NewIntResult(int64(len(m.lists[key])), nil)
}

func (m *mockRedisClient) LPush(_ context.Context, key string, values ...interface{}) *redis.IntCmd {
	if m.down.Load() {
		return redis.NewIntResult(0, errMockRedisUnavailable)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	list := m.lists[key]
	for _, val := range values {
		list = append([]string{normalizeRedisValue(val)}, list...)
	}
	m.lists[key] = list
	return redis.NewIntResult(int64(len(list)), nil)
}

func (m *mockRedisClient) BRPop(ctx context.Context, timeout time.Duration, keys ...string) *redis.StringSliceCmd {
	if m.down.Load() {
		return redis.NewStringSliceResult(nil, errMockRedisUnavailable)
	}

	deadline := time.Now().Add(timeout)
	for {
		m.mu.Lock()
		for _, key := range keys {
			list := m.lists[key]
			if len(list) == 0 {
				continue
			}
			value := list[len(list)-1]
			m.lists[key] = list[:len(list)-1]
			m.mu.Unlock()
			return redis.NewStringSliceResult([]string{key, value}, nil)
		}
		m.mu.Unlock()

		if timeout <= 0 || time.Now().After(deadline) {
			return redis.NewStringSliceResult(nil, redis.Nil)
		}
		select {
		case <-ctx.Done():
			return redis.NewStringSliceResult(nil, ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
		if m.down.Load() {
			return redis.NewStringSliceResult(nil, errMockRedisUnavailable)
		}
	}
}

func (m *mockRedisClient) SAdd(_ context.Context, key string, members ...interface{}) *redis.IntCmd {
	if m.down.Load() {
		return redis.NewIntResult(0, errMockRedisUnavailable)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	set, ok := m.sets[key]
	if !ok {
		set = make(map[string]struct{})
		m.sets[key] = set
	}

	var added int64
	for _, member := range members {
		s := normalizeRedisValue(member)
		if _, exists := set[s]; exists {
			continue
		}
		set[s] = struct{}{}
		added++
	}
	return redis.NewIntResult(added, nil)
}

func (m *mockRedisClient) SRem(_ context.Context, key string, members ...interface{}) *redis.IntCmd {
	if m.down.Load() {
		return redis.NewIntResult(0, errMockRedisUnavailable)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	set, ok := m.sets[key]
	if !ok {
		return redis.NewIntResult(0, nil)
	}

	var removed int64
	for _, member := range members {
		s := normalizeRedisValue(member)
		if _, exists := set[s]; !exists {
			continue
		}
		delete(set, s)
		removed++
	}
	return redis.NewIntResult(removed, nil)
}

func (m *mockRedisClient) Expire(_ context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	if m.down.Load() {
		return redis.NewBoolResult(false, errMockRedisUnavailable)
	}
	_ = key
	_ = expiration
	return redis.NewBoolResult(true, nil)
}

func (m *mockRedisClient) ZAdd(_ context.Context, key string, members ...redis.Z) *redis.IntCmd {
	if m.down.Load() {
		return redis.NewIntResult(0, errMockRedisUnavailable)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing := m.zsets[key]
	var added int64
	for _, member := range members {
		normalized := normalizeRedisValue(member.Member)
		found := false
		for i := range existing {
			if existing[i].member == normalized {
				existing[i].score = member.Score
				found = true
				break
			}
		}
		if found {
			continue
		}
		existing = append(existing, mockZMember{member: normalized, score: member.Score})
		added++
	}
	m.zsets[key] = existing
	return redis.NewIntResult(added, nil)
}

func (m *mockRedisClient) ZCard(_ context.Context, key string) *redis.IntCmd {
	if m.down.Load() {
		return redis.NewIntResult(0, errMockRedisUnavailable)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return redis.NewIntResult(int64(len(m.zsets[key])), nil)
}

func (m *mockRedisClient) ZPopMin(_ context.Context, key string, count ...int64) *redis.ZSliceCmd {
	if m.down.Load() {
		return redis.NewZSliceCmdResult(nil, errMockRedisUnavailable)
	}

	popCount := int64(1)
	if len(count) > 0 && count[0] > 0 {
		popCount = count[0]
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	set := m.zsets[key]
	if len(set) == 0 {
		return redis.NewZSliceCmdResult(nil, nil)
	}

	sort.Slice(set, func(i, j int) bool {
		if set[i].score == set[j].score {
			return set[i].member < set[j].member
		}
		return set[i].score < set[j].score
	})

	if int64(len(set)) < popCount {
		popCount = int64(len(set))
	}

	out := make([]redis.Z, 0, popCount)
	for i := int64(0); i < popCount; i++ {
		out = append(out, redis.Z{
			Member: set[i].member,
			Score:  set[i].score,
		})
	}

	m.zsets[key] = set[popCount:]
	return redis.NewZSliceCmdResult(out, nil)
}

func (m *mockRedisClient) ZPopMax(_ context.Context, key string, count ...int64) *redis.ZSliceCmd {
	if m.down.Load() {
		return redis.NewZSliceCmdResult(nil, errMockRedisUnavailable)
	}

	popCount := int64(1)
	if len(count) > 0 && count[0] > 0 {
		popCount = count[0]
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	set := m.zsets[key]
	if len(set) == 0 {
		return redis.NewZSliceCmdResult(nil, nil)
	}

	sort.Slice(set, func(i, j int) bool {
		if set[i].score == set[j].score {
			return set[i].member < set[j].member
		}
		return set[i].score > set[j].score
	})

	if int64(len(set)) < popCount {
		popCount = int64(len(set))
	}

	out := make([]redis.Z, 0, popCount)
	for i := int64(0); i < popCount; i++ {
		out = append(out, redis.Z{
			Member: set[i].member,
			Score:  set[i].score,
		})
	}

	m.zsets[key] = set[popCount:]
	return redis.NewZSliceCmdResult(out, nil)
}

func normalizeRedisValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprint(val)
	}
}

func requireRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("GOCLAW_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("redis is not available at %s: %v", addr, err)
	}

	t.Cleanup(func() {
		_ = client.Close()
	})

	return client
}

func requireRedisClientTB(tb testing.TB) *redis.Client {
	tb.Helper()

	addr := os.Getenv("GOCLAW_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		tb.Skipf("redis is not available at %s: %v", addr, err)
	}

	tb.Cleanup(func() {
		_ = client.Close()
	})

	return client
}

func uniqueKeyPrefix(prefix string) string {
	return fmt.Sprintf("goclaw:test:%s:%d:", prefix, time.Now().UnixNano())
}
