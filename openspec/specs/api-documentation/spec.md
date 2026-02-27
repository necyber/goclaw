# api-documentation Specification

## Purpose
Migrated from legacy OpenSpec format. Legacy narrative is retained in Notes.

## Requirements

### Requirement: Legacy specification baseline
The system SHALL preserve and implement the legacy behavior documented for api-documentation.

#### Scenario: Baseline conformance
- **WHEN** implementations reference this capability
- **THEN** they MUST conform to the legacy details captured in the notes section.

## Notes

# 瑙勮寖锛欰PI 鏂囨。

## 姒傝堪

涓?HTTP API 鎻愪緵瀹屾暣鐨勬枃妗ｏ紝鍖呮嫭 OpenAPI/Swagger 瑙勮寖銆佷氦浜掑紡鏂囨。鐣岄潰鍜屼娇鐢ㄧず渚嬨€?
## OpenAPI 瑙勮寖

### 瑙勮寖鐗堟湰
- 浣跨敤 OpenAPI 3.0.3 鎴栨洿楂樼増鏈?- 閬靛惊 OpenAPI 瑙勮寖鏍囧噯

### 瑙勮寖鍐呭

**鍩烘湰淇℃伅**:
```yaml
openapi: 3.0.3
info:
  title: Goclaw API
  description: 鍒嗗竷寮忓浠ｇ悊缂栨帓寮曟搸 HTTP API
  version: 0.1.0
  contact:
    name: Goclaw Team
  license:
    name: MIT
servers:
  - url: http://localhost:8080/api/v1
    description: 鏈湴寮€鍙戞湇鍔″櫒
```

**鏍囩鍒嗙粍**:
- `workflows`: 宸ヤ綔娴佺鐞?- `health`: 鍋ュ悍鐩戞帶

### 绔偣鏂囨。瑕佹眰

姣忎釜绔偣蹇呴』鍖呭惈锛?- 瀹屾暣鐨勬弿杩?- 璇锋眰鍙傛暟锛堣矾寰勩€佹煡璇€佽姹備綋锛?- 鍝嶅簲绀轰緥锛堟垚鍔熷拰閿欒锛?- HTTP 鐘舵€佺爜璇存槑
- 鏁版嵁妯″瀷瀹氫箟

### 鏁版嵁妯″瀷锛圫chemas锛?
瀹氫箟鎵€鏈夎姹傚拰鍝嶅簲鐨勬暟鎹粨鏋勶細
- `WorkflowRequest`
- `WorkflowResponse`
- `TaskDefinition`
- `TaskStatus`
- `ErrorResponse`
- `HealthResponse`
- `ReadyResponse`

## 浜や簰寮忔枃妗?
### Swagger UI

**绔偣**: `GET /docs`

**鍔熻兘**:
- 鎻愪緵浜や簰寮?API 鏂囨。鐣岄潰
- 鏀寔鍦ㄧ嚎娴嬭瘯 API 绔偣
- 鏄剧ず璇锋眰/鍝嶅簲绀轰緥
- 鑷姩浠?OpenAPI 瑙勮寖鐢熸垚

**瀹炵幇鏂瑰紡**:
- 浣跨敤 `swaggo/swag` 鐢熸垚 OpenAPI 瑙勮寖
- 浣跨敤 `swaggo/http-swagger` 鎻愪緵 Swagger UI
- 鎴栦娇鐢ㄩ潤鎬佹枃浠舵墭绠?Swagger UI

### ReDoc锛堝彲閫夛級

**绔偣**: `GET /redoc`

**鍔熻兘**:
- 鎻愪緵鍙︿竴绉嶆枃妗ｇ晫闈㈤€夋嫨
- 鏇撮€傚悎闃呰鍜屾墦鍗?- 鍝嶅簲寮忚璁?
## 浠ｇ爜娉ㄨВ

### Swag 娉ㄨВ绀轰緥

鍦ㄥ鐞嗗櫒鍑芥暟涓婃坊鍔犳敞瑙ｏ細

```go
// SubmitWorkflow godoc
// @Summary 鎻愪氦宸ヤ綔娴?// @Description 鎻愪氦鏂扮殑宸ヤ綔娴佽繘琛屾墽琛?// @Tags workflows
// @Accept json
// @Produce json
// @Param workflow body WorkflowRequest true "宸ヤ綔娴佸畾涔?
// @Success 201 {object} WorkflowResponse
// @Failure 400 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /workflows [post]
func (h *WorkflowHandler) SubmitWorkflow(w http.ResponseWriter, r *http.Request) {
    // 瀹炵幇
}
```

### 鏁版嵁妯″瀷娉ㄨВ

```go
// WorkflowRequest 宸ヤ綔娴佹彁浜よ姹?type WorkflowRequest struct {
    // 宸ヤ綔娴佸悕绉?    Name string `json:"name" example:"my-workflow"`
    // 浠诲姟鍒楄〃
    Tasks []TaskDefinition `json:"tasks"`
    // 鍏冩暟鎹?    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

## 鏂囨。鐢熸垚

### 鐢熸垚鍛戒护

```bash
# 瀹夎 swag
go install github.com/swaggo/swag/cmd/swag@latest

# 鐢熸垚鏂囨。
swag init -g cmd/goclaw/main.go -o docs/swagger

# 杈撳嚭鏂囦欢
# - docs/swagger/swagger.json
# - docs/swagger/swagger.yaml
# - docs/swagger/docs.go
```

### 闆嗘垚鍒版湇鍔″櫒

```go
import (
    httpSwagger "github.com/swaggo/http-swagger"
    _ "goclaw/docs/swagger" // 瀵煎叆鐢熸垚鐨勬枃妗?)

// 娉ㄥ唽璺敱
router.Get("/docs/*", httpSwagger.WrapHandler)
```

## 鏂囦欢缁撴瀯

```
docs/
鈹溾攢鈹€ swagger/
鈹?  鈹溾攢鈹€ swagger.json    # OpenAPI JSON 瑙勮寖
鈹?  鈹溾攢鈹€ swagger.yaml    # OpenAPI YAML 瑙勮寖
鈹?  鈹斺攢鈹€ docs.go         # 鐢熸垚鐨?Go 浠ｇ爜
鈹斺攢鈹€ examples/
    鈹溾攢鈹€ submit_workflow.json
    鈹斺攢鈹€ workflow_response.json
```

## 浣跨敤绀轰緥

### cURL 绀轰緥

```bash
# 鎻愪氦宸ヤ綔娴?curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "example-workflow",
    "tasks": [
      {"id": "task-1", "dependencies": []},
      {"id": "task-2", "dependencies": ["task-1"]}
    ]
  }'

# 鏌ヨ宸ヤ綔娴佺姸鎬?curl http://localhost:8080/api/v1/workflows/{workflow_id}
```

### Go 瀹㈡埛绔ず渚?
```go
// 鎻愪氦宸ヤ綔娴?req := &WorkflowRequest{
    Name: "example-workflow",
    Tasks: []TaskDefinition{
        {ID: "task-1", Dependencies: []string{}},
        {ID: "task-2", Dependencies: []string{"task-1"}},
    },
}

resp, err := http.Post(
    "http://localhost:8080/api/v1/workflows",
    "application/json",
    bytes.NewBuffer(jsonData),
)
```

## 渚濊禆

- `github.com/swaggo/swag` - OpenAPI 瑙勮寖鐢熸垚
- `github.com/swaggo/http-swagger` - Swagger UI 闆嗘垚

## 娴嬭瘯瑕佹眰

### 鏂囨。楠岃瘉
- OpenAPI 瑙勮寖鏈夋晥鎬?- 鎵€鏈夌鐐归兘鏈夋枃妗?- 绀轰緥鏁版嵁姝ｇ‘
- 鏁版嵁妯″瀷瀹屾暣

### 鍙闂€ф祴璇?- `/docs` 绔偣鍙闂?- Swagger UI 姝ｅ父鍔犺浇
- 鍙互鎵ц娴嬭瘯璇锋眰

## 楠屾敹鏍囧噯

- [ ] OpenAPI 瑙勮寖瀹屾暣涓旀湁鏁?- [ ] 鎵€鏈夌鐐归兘鏈夎缁嗘枃妗?- [ ] Swagger UI 鍙闂苟姝ｅ父宸ヤ綔
- [ ] 鍖呭惈璇锋眰/鍝嶅簲绀轰緥
- [ ] 鏁版嵁妯″瀷瀹氫箟瀹屾暣
- [ ] 鎻愪緵浣跨敤绀轰緥锛坈URL銆丟o锛?- [ ] 鏂囨。鑷姩鐢熸垚娴佺▼姝ｅ父
- [ ] 鏂囨。涓庡疄闄?API 涓€鑷?

