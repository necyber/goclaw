# Distributed Lane Guide

This guide explains how to run GoClaw with Redis-backed lanes and Redis-backed signal delivery.

## 1. What Distributed Lane Enables

- Queue backend can switch from in-memory to Redis.
- Signal bus can switch from local channels to Redis Pub/Sub.
- Engine falls back to local mode when Redis is unavailable.

## 2. Required Configuration

Set these blocks in your config file.

```yaml
orchestration:
  queue:
    type: redis
    size: 10000

redis:
  enabled: true
  address: "127.0.0.1:6379"
  password: ""
  db: 0
  max_retries: 3
  pool_size: 10
  min_idle_conns: 2
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  sentinel:
    enabled: false
    master_name: "mymaster"
    addresses: []

signal:
  mode: redis
  buffer_size: 16
  channel_prefix: "goclaw:signal:"
```

## 3. Redis Lane Options

- `orchestration.queue.type`: `memory` or `redis`.
- `orchestration.queue.size`: queue capacity for the default lane.
- `redis.enabled`: enables Redis bootstrap for lane/signal.
- `redis.address`: Redis endpoint (`host:port`).
- `redis.db`: Redis database index.
- `redis.max_retries`: Redis command retry limit.
- `redis.pool_size`: max connection pool size.
- `redis.min_idle_conns`: minimum idle connections.
- `redis.sentinel.*`: Sentinel failover settings.

## 4. Signal Bus Options

- `signal.mode`: `local` or `redis`.
- `signal.buffer_size`: per-task signal channel buffer.
- `signal.channel_prefix`: Redis Pub/Sub channel prefix.

Supported patterns:

- `steer`: runtime parameter steering.
- `interrupt`: graceful/forced task interruption.
- `collect`: fan-in result collection.

## 5. Runtime Behavior and Fallback

- If Redis init fails at startup, queue and signal automatically degrade to local mode.
- Startup logs include effective runtime mode:
  - `queue_type=redis` or `memory(fallback)`
  - `signal_mode=redis` or `local(fallback)`
  - `redis_connected=true|false`

## 6. Docker Compose Deployment

Use `docker-compose.yml` in the repo root. It includes Redis and wires these env vars:

- `GOCLAW_ORCHESTRATION_QUEUE_TYPE=redis`
- `GOCLAW_REDIS_ENABLED=true`
- `GOCLAW_REDIS_ADDRESS=redis:6379`
- `GOCLAW_SIGNAL_MODE=redis`

Start stack:

```bash
docker compose up -d
```

Check service status:

```bash
docker compose ps
docker compose logs -f goclaw
```

## 7. Validation Checklist

- Redis is reachable from GoClaw container.
- Startup log shows `redis_connected=true`.
- Queue mode is `redis` (not `memory(fallback)`).
- Signal mode is `redis` (not `local(fallback)`).
- `/metrics` exports lane/signal metrics.
