package lane

import "fmt"

// LaneType describes the backend type for a lane.
type LaneType string

const (
	// LaneTypeMemory uses the in-memory ChannelLane.
	LaneTypeMemory LaneType = "memory"
	// LaneTypeRedis uses the Redis-backed lane.
	LaneTypeRedis LaneType = "redis"
)

// LaneSpec describes how to construct a lane of a given type.
type LaneSpec struct {
	// Type selects the backend implementation.
	Type LaneType

	// Memory holds configuration for the in-memory lane.
	Memory *Config

	// Redis holds configuration for the Redis-backed lane.
	Redis *RedisConfig

	// Fallback optionally overrides the local lane config for Redis fallback.
	Fallback *Config

	// FallbackConfig configures the Redis fallback behavior.
	FallbackConfig *FallbackConfig
}

// Name returns the configured lane name, if available.
func (s *LaneSpec) Name() string {
	switch s.Type {
	case LaneTypeRedis:
		if s.Redis != nil {
			return s.Redis.Name
		}
	default:
		if s.Memory != nil {
			return s.Memory.Name
		}
	}
	return ""
}

// Validate ensures the spec is complete and internally consistent.
func (s *LaneSpec) Validate() error {
	if s == nil {
		return fmt.Errorf("lane spec cannot be nil")
	}

	if s.Type == "" {
		switch {
		case s.Redis != nil:
			s.Type = LaneTypeRedis
		case s.Memory != nil:
			s.Type = LaneTypeMemory
		}
	}

	switch s.Type {
	case LaneTypeMemory:
		if s.Memory == nil {
			return fmt.Errorf("memory lane config cannot be nil")
		}
		return s.Memory.Validate()
	case LaneTypeRedis:
		if s.Redis == nil {
			return fmt.Errorf("redis lane config cannot be nil")
		}
		if err := s.Redis.Validate(); err != nil {
			return err
		}
		if s.Fallback != nil && s.Fallback.Name == "" {
			return fmt.Errorf("fallback lane name cannot be empty")
		}
		return nil
	default:
		return fmt.Errorf("unsupported lane type: %s", s.Type)
	}
}

func (s *LaneSpec) fallbackConfig() *Config {
	if s.Fallback != nil {
		return s.Fallback
	}
	if s.Redis == nil {
		return nil
	}
	return &Config{
		Name:           s.Redis.Name,
		Capacity:       s.Redis.Capacity,
		MaxConcurrency: s.Redis.MaxConcurrency,
		Backpressure:   s.Redis.Backpressure,
		RedirectLane:   s.Redis.RedirectLane,
		EnablePriority: s.Redis.EnablePriority,
	}
}
