# http-server-core Specification

## Purpose
Migrated from legacy OpenSpec format. Legacy narrative is retained in Notes.

## Requirements

### Requirement: Legacy specification baseline
The system SHALL preserve and implement the legacy behavior documented for http-server-core.

#### Scenario: Baseline conformance
- **WHEN** implementations reference this capability
- **THEN** they MUST conform to the legacy details captured in the notes section.

## Notes

# 瑙勮寖锛欻TTP 鏈嶅姟鍣ㄦ牳蹇?
## 姒傝堪

瀹炵幇 Goclaw 鐨勬牳蹇?HTTP 鏈嶅姟鍣ㄥ熀纭€璁炬柦锛屽寘鎷湇鍔″櫒鐢熷懡鍛ㄦ湡绠＄悊銆佽矾鐢辩郴缁熴€佷腑闂翠欢閾惧拰浼橀泤鍏抽棴鏈哄埗銆?
## 鍔熻兘闇€姹?
### 1. HTTP 鏈嶅姟鍣ㄥ垵濮嬪寲

**杈撳叆**:
- 閰嶇疆瀵硅薄锛堢鍙ｃ€佽秴鏃躲€乀LS 璁剧疆绛夛級
- 寮曟搸瀹炰緥寮曠敤

**杈撳嚭**:
- 宸查厤缃殑 HTTP 鏈嶅姟鍣ㄥ疄渚?
**琛屼负**:
- 浠庨厤缃姞杞芥湇鍔″櫒鍙傛暟锛堢洃鍚湴鍧€銆佺鍙ｃ€佽秴鏃讹級
- 鍒濆鍖栬矾鐢卞櫒
- 娉ㄥ唽涓棿浠堕摼
- 缁戝畾鍒版寚瀹氱鍙?- 鏀寔 HTTP 鍜?HTTPS锛堝彲閫夛級

### 2. 璺敱绯荤粺

**闇€姹?*:
- 鏀寔 RESTful 璺敱妯″紡锛圙ET銆丳OST銆丳UT銆丏ELETE銆丳ATCH锛?- 璺緞鍙傛暟鎻愬彇锛堝 `/workflows/{id}`锛?- 鏌ヨ鍙傛暟瑙ｆ瀽
- 璺敱鍒嗙粍锛堝 `/api/v1/...`锛?- 404 鍜?405 澶勭悊

**鎺ㄨ崘瀹炵幇**:
- 浣跨敤 `chi` 璺敱鍣紙杞婚噺銆佹爣鍑嗗簱鍏煎锛?- 鎴栦娇鐢ㄦ爣鍑嗗簱 `net/http` 鐨?`ServeMux`锛圙o 1.22+锛?
### 3. 涓棿浠堕摼

**蹇呴渶涓棿浠?*:
- **鏃ュ織涓棿浠?*: 璁板綍璇锋眰鏂规硶銆佽矾寰勩€佺姸鎬佺爜銆佸搷搴旀椂闂?- **鎭㈠涓棿浠?*: 鎹曡幏 panic锛岃繑鍥?500 閿欒
- **CORS 涓棿浠?*: 閰嶇疆璺ㄥ煙璧勬簮鍏变韩绛栫暐
- **璇锋眰 ID 涓棿浠?*: 涓烘瘡涓姹傜敓鎴愬敮涓€ ID
- **瓒呮椂涓棿浠?*: 璁剧疆璇锋眰澶勭悊瓒呮椂

**鍙€変腑闂翠欢**:
- 閫熺巼闄愬埗
- 璁よ瘉/鎺堟潈锛堜负鏈潵鎵╁睍棰勭暀锛?- 鍘嬬缉锛坓zip锛?
### 4. 浼橀泤鍏抽棴

**闇€姹?*:
- 鐩戝惉绯荤粺淇″彿锛圫IGINT銆丼IGTERM锛?- 鍋滄鎺ュ彈鏂拌繛鎺?- 绛夊緟鐜版湁璇锋眰瀹屾垚锛堝甫瓒呮椂锛?- 娓呯悊璧勬簮
- 璁板綍鍏抽棴浜嬩欢

**瀹炵幇**:
```go
// 浼唬鐮?ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
server.Shutdown(ctx)
```

### 5. 閿欒澶勭悊

**鏍囧噯閿欒鍝嶅簲鏍煎紡**:
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "浜虹被鍙鐨勯敊璇秷鎭?,
    "details": {},
    "request_id": "uuid"
  }
}
```

**HTTP 鐘舵€佺爜鏄犲皠**:
- 400: 璇锋眰鍙傛暟閿欒
- 404: 璧勬簮鏈壘鍒?- 405: 鏂规硶涓嶅厑璁?- 500: 鍐呴儴鏈嶅姟鍣ㄩ敊璇?- 503: 鏈嶅姟涓嶅彲鐢?
## 閰嶇疆

### 閰嶇疆缁撴瀯

```yaml
server:
  http:
    enabled: true
    host: "0.0.0.0"
    port: 8080
    read_timeout: 30s
    write_timeout: 30s
    idle_timeout: 120s
    shutdown_timeout: 30s
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "PATCH"]
    allowed_headers: ["Content-Type", "Authorization"]
    max_age: 3600
```

### 閰嶇疆楠岃瘉

- `port` 蹇呴』鍦?1-65535 鑼冨洿鍐?- 瓒呮椂鍊煎繀椤讳负姝ｆ暟
- CORS 閰嶇疆蹇呴』鏈夋晥

## 鎺ュ彛璁捐

### Server 鎺ュ彛

```go
type Server interface {
    Start() error
    Shutdown(ctx context.Context) error
    RegisterRoutes(router chi.Router)
}
```

### HTTPServer 瀹炵幇

```go
type HTTPServer struct {
    config *config.ServerConfig
    engine *engine.Engine
    server *http.Server
    router chi.Router
    logger *logger.Logger
}

func NewHTTPServer(cfg *config.ServerConfig, eng *engine.Engine, log *logger.Logger) *HTTPServer
```

## 鏂囦欢缁撴瀯

```
pkg/api/
鈹溾攢鈹€ server.go           # HTTPServer 瀹炵幇
鈹溾攢鈹€ middleware/
鈹?  鈹溾攢鈹€ logger.go       # 鏃ュ織涓棿浠?鈹?  鈹溾攢鈹€ recovery.go     # 鎭㈠涓棿浠?鈹?  鈹溾攢鈹€ cors.go         # CORS 涓棿浠?鈹?  鈹溾攢鈹€ request_id.go   # 璇锋眰 ID 涓棿浠?鈹?  鈹斺攢鈹€ timeout.go      # 瓒呮椂涓棿浠?鈹溾攢鈹€ response/
鈹?  鈹溾攢鈹€ json.go         # JSON 鍝嶅簲杈呭姪鍑芥暟
鈹?  鈹斺攢鈹€ error.go        # 閿欒鍝嶅簲鏍煎紡鍖?鈹斺攢鈹€ router.go           # 璺敱娉ㄥ唽
```

## 渚濊禆

- `github.com/go-chi/chi/v5` - HTTP 璺敱鍣?- `github.com/go-chi/cors` - CORS 涓棿浠?- 鐜版湁鐨?`pkg/logger` - 鏃ュ織璁板綍
- 鐜版湁鐨?`config` - 閰嶇疆绠＄悊
- 鐜版湁鐨?`pkg/engine` - 寮曟搸鎺ュ彛

## 娴嬭瘯瑕佹眰

### 鍗曞厓娴嬭瘯
- 鏈嶅姟鍣ㄥ垵濮嬪寲
- 涓棿浠跺姛鑳?- 閿欒鍝嶅簲鏍煎紡鍖?- 閰嶇疆楠岃瘉

### 闆嗘垚娴嬭瘯
- 鏈嶅姟鍣ㄥ惎鍔ㄥ拰鍏抽棴
- 浼橀泤鍏抽棴娴佺▼
- 涓棿浠堕摼鎵ц椤哄簭
- 鍩烘湰璺敱鍔熻兘

## 楠屾敹鏍囧噯

- [ ] HTTP 鏈嶅姟鍣ㄥ彲浠ュ湪閰嶇疆鐨勭鍙ｄ笂鍚姩
- [ ] 鎵€鏈変腑闂翠欢姝ｇ‘搴旂敤鍒拌姹傞摼
- [ ] 浼橀泤鍏抽棴鍦ㄨ秴鏃跺唴瀹屾垚
- [ ] 閿欒鍝嶅簲鏍煎紡涓€鑷?- [ ] CORS 澶存纭缃?- [ ] 璇锋眰鏃ュ織鍖呭惈鎵€鏈夊繀闇€瀛楁
- [ ] 鍗曞厓娴嬭瘯瑕嗙洊鐜?> 80%
- [ ] 闆嗘垚娴嬭瘯閫氳繃

