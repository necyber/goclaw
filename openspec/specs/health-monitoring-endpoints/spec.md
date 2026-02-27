# health-monitoring-endpoints Specification

## Purpose
Migrated from legacy OpenSpec format. Legacy narrative is retained in Notes.

## Requirements

### Requirement: Legacy specification baseline
The system SHALL preserve and implement the legacy behavior documented for health-monitoring-endpoints.

#### Scenario: Baseline conformance
- **WHEN** implementations reference this capability
- **THEN** they MUST conform to the legacy details captured in the notes section.

## Notes

# 瑙勮寖锛氬仴搴风洃鎺х鐐?
## 姒傝堪

瀹炵幇鐢ㄤ簬绯荤粺鍋ュ悍妫€鏌ュ拰灏辩华鐘舵€佺洃鎺х殑绔偣锛屾敮鎸?Kubernetes 绛夊鍣ㄧ紪鎺掑钩鍙扮殑鍋ュ悍鎺㈡祴銆?
## API 绔偣

### 1. 鍋ュ悍妫€鏌?
**绔偣**: `GET /health`

**鐢ㄩ€?*: 妫€鏌ユ湇鍔℃槸鍚﹀瓨娲伙紙liveness probe锛?
**鍝嶅簲** (200 OK):
```json
{
  "status": "healthy",
  "timestamp": "2026-02-24T10:00:00Z"
}
```

**鍝嶅簲** (503 Service Unavailable):
```json
{
  "status": "unhealthy",
  "timestamp": "2026-02-24T10:00:00Z",
  "error": "鏈嶅姟涓嶅彲鐢?
}
```

**妫€鏌ラ」**:
- HTTP 鏈嶅姟鍣ㄦ鍦ㄨ繍琛?- 鍩烘湰鍝嶅簲鑳藉姏

**鐗圭偣**:
- 杞婚噺绾э紝蹇€熷搷搴?- 涓嶆鏌ヤ緷璧栭」
- 鐢ㄤ簬鍒ゆ柇鏄惁闇€瑕侀噸鍚湇鍔?
### 2. 灏辩华妫€鏌?
**绔偣**: `GET /ready`

**鐢ㄩ€?*: 妫€鏌ユ湇鍔℃槸鍚﹀噯澶囧ソ鎺ユ敹娴侀噺锛坮eadiness probe锛?
**鍝嶅簲** (200 OK):
```json
{
  "status": "ready",
  "timestamp": "2026-02-24T10:00:00Z",
  "checks": {
    "engine": "ok",
    "storage": "ok"
  }
}
```

**鍝嶅簲** (503 Service Unavailable):
```json
{
  "status": "not_ready",
  "timestamp": "2026-02-24T10:00:00Z",
  "checks": {
    "engine": "ok",
    "storage": "failed"
  },
  "error": "瀛樺偍鏈氨缁?
}
```

**妫€鏌ラ」**:
- 寮曟搸鐘舵€侊紙鏄惁宸插垵濮嬪寲锛?- 瀛樺偍杩炴帴锛堝鏋滃凡閰嶇疆锛?- 鍏抽敭渚濊禆椤瑰彲鐢ㄦ€?
**鐗圭偣**:
- 妫€鏌ヤ緷璧栭」鐘舵€?- 澶辫触鏃舵湇鍔′笉鎺ユ敹娴侀噺
- 鐢ㄤ簬婊氬姩鏇存柊鍜岃礋杞藉潎琛?
### 3. 璇︾粏鐘舵€侊紙鍙€夛級

**绔偣**: `GET /status`

**鐢ㄩ€?*: 鑾峰彇璇︾粏鐨勭郴缁熺姸鎬佷俊鎭?
**鍝嶅簲** (200 OK):
```json
{
  "status": "running",
  "version": "0.1.0",
  "uptime": "2h30m15s",
  "timestamp": "2026-02-24T10:00:00Z",
  "components": {
    "engine": {
      "status": "running",
      "workflows_active": 5,
      "workflows_total": 100
    },
    "lanes": {
      "status": "ok",
      "total_lanes": 3,
      "active_workers": 10
    },
    "storage": {
      "status": "connected",
      "type": "memory"
    }
  },
  "system": {
    "goroutines": 50,
    "memory_mb": 128
  }
}
```

**鐗圭偣**:
- 鎻愪緵璇︾粏鐨勮繍琛屾椂淇℃伅
- 鐢ㄤ簬鐩戞帶鍜岃皟璇?- 鍙兘鍖呭惈鏁忔劅淇℃伅锛岃€冭檻璁块棶鎺у埗

## 瀹炵幇璁捐

### HealthChecker 鎺ュ彛

```go
type HealthChecker interface {
    Check(ctx context.Context) error
    Name() string
}
```

### 鍐呯疆妫€鏌ュ櫒

```go
// 寮曟搸鍋ュ悍妫€鏌?type EngineHealthChecker struct {
    engine *engine.Engine
}

// 瀛樺偍鍋ュ悍妫€鏌?type StorageHealthChecker struct {
    storage storage.Storage
}
```

### HealthHandler 瀹炵幇

```go
type HealthHandler struct {
    checkers []HealthChecker
    startTime time.Time
    version string
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request)
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request)
func (h *HealthHandler) Status(w http.ResponseWriter, r *http.Request)
```

## 鏂囦欢缁撴瀯

```
pkg/api/handlers/
鈹溾攢鈹€ health.go           # 鍋ュ悍妫€鏌ュ鐞嗗櫒
鈹溾攢鈹€ health_test.go      # 鍗曞厓娴嬭瘯
鈹斺攢鈹€ checkers/
    鈹溾攢鈹€ engine.go       # 寮曟搸妫€鏌ュ櫒
    鈹斺攢鈹€ storage.go      # 瀛樺偍妫€鏌ュ櫒
```

## Kubernetes 闆嗘垚绀轰緥

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2
```

## 娴嬭瘯瑕佹眰

### 鍗曞厓娴嬭瘯
- 鍋ュ悍妫€鏌ラ€昏緫
- 灏辩华妫€鏌ラ€昏緫
- 妫€鏌ュ櫒鎺ュ彛瀹炵幇
- 鍝嶅簲鏍煎紡

### 闆嗘垚娴嬭瘯
- 绔偣鍙闂€?- 鐘舵€佺爜姝ｇ‘鎬?- 渚濊禆椤瑰け璐ュ満鏅?- 瓒呮椂澶勭悊

## 楠屾敹鏍囧噯

- [ ] `/health` 绔偣濮嬬粓蹇€熷搷搴?- [ ] `/ready` 绔偣姝ｇ‘鍙嶆槧渚濊禆椤圭姸鎬?- [ ] `/status` 绔偣鎻愪緵璇︾粏淇℃伅
- [ ] 鍝嶅簲鏍煎紡涓€鑷?- [ ] 鏀寔瓒呮椂鎺у埗
- [ ] 鍗曞厓娴嬭瘯瑕嗙洊鐜?> 80%
- [ ] 涓?Kubernetes 鎺㈡祴鍏煎

