# Hybrid Memory System Guide

Goclaw's hybrid memory system provides intelligent memory management for agent orchestration, combining vector-based semantic search, BM25 full-text retrieval, and FSRS-6 spaced-repetition decay.

## Architecture

```
┌─────────────────────────────────────────────┐
│                 Memory Hub                   │
│  (Memorize / Retrieve / Forget / Stats)      │
├──────────┬──────────┬───────────────────────┤
│  Vector  │  BM25    │   Hybrid Retriever    │
│  Index   │  Index   │   (RRF Fusion)        │
├──────────┴──────────┴───────────────────────┤
│            Tiered Storage                    │
│   L1 (LRU Cache) → L2 (Badger Persistence)  │
├─────────────────────────────────────────────┤
│         FSRS-6 Decay Manager                 │
│   (Background strength decay loop)           │
└─────────────────────────────────────────────┘
```

### Components

- **Memory Hub** — Central API coordinating all subsystems
- **Vector Index** — Cosine similarity search over embedding vectors
- **BM25 Index** — Full-text search with TF-IDF scoring (supports CJK)
- **Hybrid Retriever** — Reciprocal Rank Fusion (RRF) combining both indexes
- **Tiered Storage** — L1 LRU cache backed by L2 Badger persistence
- **Decay Manager** — FSRS-6 algorithm for automatic memory strength decay

## Configuration

Enable the memory system in your config file:

```yaml
memory:
  enabled: true
  vector_dimension: 768          # Must match your embedding model output
  vector_weight: 0.7             # Weight for vector retrieval in hybrid search
  bm25_weight: 0.3               # Weight for BM25 retrieval in hybrid search
  l1_cache_size: 1000            # Max entries in L1 LRU cache
  forget_threshold: 0.1          # Strength below which entries are auto-deleted
  decay_interval: 1h             # How often the decay loop runs
  default_stability: 24.0        # Initial stability for new entries (hours)
  storage_path: "./data/memory"  # Directory for memory data persistence

  bm25:
    k1: 1.5                      # Term frequency saturation (1.2-2.0 typical)
    b: 0.75                      # Document length normalization (0.0-1.0)

  hnsw:
    m: 16                        # Bi-directional links per element
    ef_construction: 200         # Candidate list size during construction
    ef_search: 100               # Candidate list size during search
```

### Configuration Reference

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `false` | Enable/disable the memory system |
| `vector_dimension` | int | `768` | Embedding vector dimension (must match your model) |
| `vector_weight` | float | `0.7` | Vector retrieval weight in hybrid mode (0.0-1.0) |
| `bm25_weight` | float | `0.3` | BM25 retrieval weight in hybrid mode (0.0-1.0) |
| `l1_cache_size` | int | `1000` | Maximum entries in L1 LRU cache |
| `forget_threshold` | float | `0.1` | Auto-delete entries with strength below this |
| `decay_interval` | duration | `1h` | Background decay loop interval |
| `default_stability` | float | `24.0` | Initial FSRS-6 stability in hours |
| `storage_path` | string | `./data/memory` | Badger DB directory for persistence |
| `bm25.k1` | float | `1.5` | BM25 term frequency saturation |
| `bm25.b` | float | `0.75` | BM25 document length normalization |

Environment variable overrides use the `GOCLAW_` prefix:
```bash
export GOCLAW_MEMORY_ENABLED=true
export GOCLAW_MEMORY_VECTOR_DIMENSION=768
export GOCLAW_MEMORY_L1_CACHE_SIZE=2000
```

## Memory Hub API

### Memorize

Store a new memory entry with content, optional embedding vector, and metadata.

```go
id, err := hub.Memorize(ctx, "session-1", "Go is a compiled language", vector, map[string]string{
    "type": "fact",
    "topic": "programming",
})
```

The entry is automatically:
- Stored in L1 cache and L2 persistent storage
- Indexed in the vector index (if vector provided)
- Indexed in the BM25 index (if content non-empty)
- Initialized with FSRS-6 strength=1.0 and configured stability

### BatchMemorize

Store multiple entries in a single call:

```go
ids, err := hub.BatchMemorize(ctx, "session-1", []memory.BatchEntry{
    {Content: "Entry 1", Vector: vec1, Metadata: meta1},
    {Content: "Entry 2", Vector: vec2, Metadata: meta2},
})
```

### Retrieve

Search for relevant memories using text, vector, or hybrid mode:

```go
results, err := hub.Retrieve(ctx, "session-1", memory.Query{
    Text:    "compiled languages",
    Mode:    "hybrid",  // "hybrid", "vector", "bm25"
    TopK:    10,
    Filters: map[string]string{"type": "fact"},
})

for _, r := range results {
    fmt.Printf("Score: %.3f Content: %s\n", r.Score, r.Entry.Content)
}
```

Retrieval modes:
- `hybrid` (default) — Combines vector and BM25 results using RRF fusion
- `vector` — Cosine similarity search only (requires query vector)
- `bm25` — Full-text search only (requires query text)

Retrieved entries automatically get a strength boost (FSRS-6 spaced repetition).

### Forget

Delete specific entries by ID:

```go
err := hub.Forget(ctx, "session-1", []string{"entry-id-1", "entry-id-2"})
```

### ForgetByThreshold

Delete all entries with strength below a threshold:

```go
deleted, err := hub.ForgetByThreshold(ctx, "session-1", 0.2)
fmt.Printf("Deleted %d weak memories\n", deleted)
```

### List

Paginated listing of all entries in a session:

```go
entries, total, err := hub.List(ctx, "session-1", 20, 0) // limit=20, offset=0
```

### Count

Get the total number of entries for a session:

```go
count, err := hub.Count(ctx, "session-1")
```

### GetStats

Get memory statistics for a session:

```go
stats, err := hub.GetStats(ctx, "session-1")
// stats.TotalEntries, stats.AverageStrength
```

### DeleteSession

Purge all entries for a session:

```go
deleted, err := hub.DeleteSession(ctx, "session-1")
```

## HTTP API Endpoints

All memory endpoints are scoped by session ID.

### Store Memory

```bash
curl -X POST http://localhost:8080/api/v1/memory/session-1 \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Go is a compiled, statically typed language",
    "vector": [0.1, 0.2, 0.3],
    "metadata": {"type": "fact", "topic": "programming"}
  }'
```

Response:
```json
{"id": "550e8400-e29b-41d4-a716-446655440000"}
```

### Query Memory

```bash
# Text search (BM25)
curl "http://localhost:8080/api/v1/memory/session-1?query=compiled+language&limit=5"

# With mode and metadata filter
curl "http://localhost:8080/api/v1/memory/session-1?query=programming&mode=bm25&metadata.type=fact"
```

Response:
```json
[
  {
    "entry": {
      "id": "550e8400-...",
      "session_id": "session-1",
      "content": "Go is a compiled, statically typed language",
      "strength": 1.0,
      "stability": 24.0,
      "created_at": "2026-02-25T10:00:00Z"
    },
    "score": 0.85
  }
]
```

### Delete Specific Entries

```bash
curl -X DELETE http://localhost:8080/api/v1/memory/session-1 \
  -H "Content-Type: application/json" \
  -d '{"ids": ["entry-id-1", "entry-id-2"]}'
```

### List Entries

```bash
curl "http://localhost:8080/api/v1/memory/session-1/list?limit=20&offset=0"
```

### Get Statistics

```bash
curl http://localhost:8080/api/v1/memory/session-1/stats
```

Response:
```json
{"total_entries": 42, "average_strength": 0.78}
```

### Delete Session

```bash
curl -X DELETE http://localhost:8080/api/v1/memory/session-1/all
```

### Delete Weak Memories

```bash
curl -X DELETE "http://localhost:8080/api/v1/memory/session-1/weak?threshold=0.2"
```

## Vector Embedding Best Practices

### Choosing Vector Dimensions

- **128 dimensions** — Lightweight, suitable for simple similarity tasks
- **384 dimensions** — Good balance of quality and performance (e.g., all-MiniLM-L6-v2)
- **768 dimensions** — High quality (e.g., all-mpnet-base-v2, BGE-base)
- **1536 dimensions** — Maximum quality (e.g., OpenAI text-embedding-3-small)

Set `vector_dimension` to match your embedding model's output dimension exactly.

### Embedding Guidelines

1. **Consistency** — Always use the same embedding model for a session. Mixing models produces meaningless similarity scores.
2. **Normalization** — Pre-normalize vectors to unit length for best cosine similarity results.
3. **Batch processing** — Use `BatchMemorize` when storing many entries to reduce overhead.
4. **Mixed mode** — You can store entries with only text (BM25-searchable) or with both text and vector (hybrid-searchable). Entries without vectors are excluded from vector search.

### When to Use Each Retrieval Mode

| Mode | Best For | Requires |
|------|----------|----------|
| `hybrid` | General-purpose retrieval | Text query (vector optional) |
| `vector` | Semantic similarity ("find similar concepts") | Query vector |
| `bm25` | Exact keyword matching | Text query |

## Performance Tuning

### L1 Cache

The L1 LRU cache stores frequently accessed entries in memory for fast retrieval.

- **Increase `l1_cache_size`** if you have high read-to-write ratio and enough RAM
- **Decrease** if memory is constrained — entries are still served from L2 (Badger)
- Cache hit promotes entries from L2 automatically

### BM25 Parameters

- **`k1` (1.0-2.0)** — Controls term frequency saturation. Higher values give more weight to term frequency. Default 1.5 works well for most cases.
- **`b` (0.0-1.0)** — Controls document length normalization. 0.75 is standard. Set to 0.0 to disable length normalization (useful for uniform-length documents).

### Hybrid Search Weights

- **`vector_weight: 0.7, bm25_weight: 0.3`** — Default. Favors semantic similarity.
- **`vector_weight: 0.5, bm25_weight: 0.5`** — Equal weight. Good when both keyword and semantic matches matter.
- **`vector_weight: 0.3, bm25_weight: 0.7`** — Favors keyword matching. Good for structured/technical content.

### Decay Tuning

- **`default_stability: 24.0`** — Entries decay to ~37% strength after 24 hours without retrieval. Increase for long-lived memories.
- **`decay_interval: 1h`** — How often the background loop runs. Shorter intervals = more responsive cleanup but more CPU.
- **`forget_threshold: 0.1`** — Entries below this strength are auto-deleted. Set to 0.0 to disable auto-deletion.

### Performance Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| Vector search (10K entries) | < 10ms | Brute-force cosine similarity |
| BM25 search (10K entries) | < 10ms | Inverted index lookup |
| Hybrid search (10K entries) | < 20ms | Parallel vector + BM25 with RRF |
| Memory footprint (10K entries, 128-dim) | < 100MB | Including indexes |
| L1 cache hit | < 1ms | In-memory LRU |

## Troubleshooting

### Memory system not starting

Check that `memory.enabled` is `true` in your config and the `storage_path` directory is writable:

```bash
# Verify config
grep -A5 "memory:" config.yaml

# Check storage path permissions
ls -la ./data/memory/
```

### Vector dimension mismatch

If you see `memory: vector dimension mismatch`, your query/entry vector length doesn't match `vector_dimension` in config. All vectors must have exactly the configured dimension.

### High memory usage

- Reduce `l1_cache_size` to limit in-memory entries
- Lower `default_stability` so entries decay faster
- Decrease `forget_threshold` to keep fewer weak entries
- Check entry count per session with the stats endpoint

### Entries disappearing unexpectedly

Entries are auto-deleted when their strength drops below `forget_threshold`. To investigate:

1. Check the current threshold: `forget_threshold` in config
2. Check entry strength via the list endpoint
3. Increase `default_stability` for slower decay
4. Set `forget_threshold: 0.0` to disable auto-deletion

### BM25 returning no results

- Verify the entry has non-empty `content` (empty content is not indexed)
- Check that the query text contains terms present in stored entries
- BM25 tokenizes by whitespace and punctuation — very short queries may not match
- CJK text is tokenized by individual characters

### Slow retrieval performance

- Check entry count — performance degrades linearly with brute-force vector search
- Ensure L1 cache is sized appropriately for your access pattern
- Use `bm25` mode if you don't need semantic search (faster for keyword queries)
- Consider reducing `TopK` to limit result set size
