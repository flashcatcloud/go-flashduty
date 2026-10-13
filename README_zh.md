# go-flashduty

[English](README.md) | 中文

[![Go Reference](https://pkg.go.dev/badge/github.com/flashcatcloud/go-flashduty.svg)](https://pkg.go.dev/github.com/flashcatcloud/go-flashduty)
[![CI](https://img.shields.io/github/actions/workflow/status/flashcatcloud/go-flashduty/ci.yml?style=flat-square&branch=main&label=CI)](https://github.com/flashcatcloud/go-flashduty/actions)
[![Release](https://img.shields.io/github/v/tag/flashcatcloud/go-flashduty?style=flat-square&color=24bfa5&label=release)](https://github.com/flashcatcloud/go-flashduty/tags)
[![License](https://img.shields.io/github/license/flashcatcloud/go-flashduty?style=flat-square&color=24bfa5)](LICENSE)

**go-flashduty** 是 [Flashduty](https://www.flashduty.com) Open API 的官方 Go SDK。Flashduty 是故障管理与值班平台；这个 SDK 让 Go 程序以强类型方式调用全部公开接口：故障、告警、协作空间、值班、状态页、监控、RUM 和 AI SRE。

[官网](https://www.flashduty.com) · [API 参考](https://docs.flashduty.com/zh/openapi/introduction) · [Go 包文档](https://pkg.go.dev/github.com/flashcatcloud/go-flashduty) · [控制台](https://console.flashcat.cloud) · [Flashduty CLI](https://github.com/flashcatcloud/flashduty-cli)

> **状态：** 类型化的接口方法由 Flashduty OpenAPI 规范生成，有单元测试覆盖，并在线上 API 上做过端到端验证。

## 安装

```bash
go get github.com/flashcatcloud/go-flashduty
```

需要 Go 1.24 及以上。

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"

	flashduty "github.com/flashcatcloud/go-flashduty"
)

func main() {
	client, err := flashduty.NewClient("YOUR_APP_KEY")
	if err != nil {
		log.Fatal(err)
	}

	list, resp, err := client.Incidents.List(context.Background(), &flashduty.ListIncidentsRequest{
		Progress:    "Triggered",
		ListOptions: flashduty.ListOptions{Limit: 20},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("request_id=%s total=%d has_next=%t\n", resp.RequestID, resp.Total, resp.HasNextPage)
	for _, inc := range list.Items {
		fmt.Printf("[%s] %s\n", inc.IncidentSeverity, inc.Title)
	}
}
```

## 认证

在[控制台](https://console.flashcat.cloud)创建 APP Key（我的 → APP Key），步骤见 [API 参考](https://docs.flashduty.com/zh/openapi/introduction)。`NewClient` 把它放在 `app_key` 查询参数里发送。

如果要用 OAuth access token 调用，改用 `NewClientWithAccessToken`：它发送 `Authorization: Bearer <token>`，其余选项相同：

```go
client, err := flashduty.NewClientWithAccessToken(accessToken)
```

## 设计

- **薄且强类型。** 每个方法只对应一次 HTTP 调用，返回 `(*T, *Response, error)`，不会在背后跨接口补数据。
- **按服务分组。** 接口按服务挂在 client 上（`client.Incidents`、`client.Alerts` 等），由 OpenAPI 规范生成。
- **可组合的传输层。** 重试、缓存、链路追踪、限流处理等通用逻辑，通过 `WithTransport` 以 `http.RoundTripper` 中间件的方式组合。
- **可读的时间戳。** 响应里的时间字段类型是 `Timestamp`（Unix 秒）或 `TimestampMilli`（毫秒），而不是裸整数，所以 JSON、日志和给 LLM 的输出都是 RFC3339 格式，需要原始 epoch 时调一个方法即可。请求字段仍是普通 `int64`。

### 选项

```go
client, err := flashduty.NewClient("YOUR_APP_KEY",
	flashduty.WithBaseURL("https://api.flashcat.cloud"),
	flashduty.WithTimeout(10*time.Second),
	flashduty.WithUserAgent("my-app/1.0"),
	flashduty.WithHTTPClient(customHTTPClient),
	flashduty.WithTransport(customRoundTripper),
	flashduty.WithLogger(myLogger),
	flashduty.WithRequestHeaders(staticHeaders),
	flashduty.WithRequestHook(func(req *http.Request) { /* 例如注入 traceparent */ }),
)
```

### 分页

列表请求内嵌 `ListOptions`。零值不会发送，因此使用服务端默认值（第 1 页，每页 20 条）。`Response.Total` 和 `Response.HasNextPage` 描述结果集：

```go
req := &flashduty.ListIncidentsRequest{ListOptions: flashduty.ListOptions{Page: 1, Limit: 100}}
for {
	list, resp, err := client.Incidents.List(ctx, req)
	if err != nil {
		return err
	}
	handle(list.Items)
	if !resp.HasNextPage {
		break
	}
	req.Page++
}
```

支持深度分页的接口会在 `Response.SearchAfterCtx` 里返回一个不透明游标；把它填回 `ListOptions.SearchAfterCtx` 即可取下一页。

### 错误与限流

```go
_, _, err := client.Incidents.Info(ctx, &flashduty.IncidentInfoRequest{IncidentID: "does-not-exist"})

var apiErr *flashduty.ErrorResponse
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.Code, apiErr.RequestID)
}

var rl *flashduty.RateLimitError
if errors.As(err, &rl) {
	time.Sleep(rl.RetryAfter)
}
```

类型化的判断函数省去字符串比较，并能穿透被包装的错误（内部用 `errors.As`）：

```go
if flashduty.IsNotFound(err) { /* ... */ }
if flashduty.IsRateLimited(err) { /* ... */ }
switch flashduty.ErrorCodeOf(err) {
case flashduty.ErrorCodeAccessDenied, flashduty.ErrorCodeUnauthorized:
	// 处理认证失败
}
```

### 时间戳

响应里的时间字段是 `Timestamp`（Unix 秒）或 `TimestampMilli`（毫秒）。序列化时输出本地时区的 RFC3339 字符串，反序列化时数字 epoch 和 RFC3339 字符串都接受，所以来回转换不丢信息。零值仍是数字 `0`（不会变成 1970 年的日期），并会被 `omitempty` 省略。

```go
inc := list.Items[0]
fmt.Println(inc.StartTime)          // 2026-05-30T14:37:11+08:00  (String / fmt / TOON)
b, _ := json.Marshal(inc.StartTime) // "2026-05-30T14:37:11+08:00"
epoch := inc.StartTime.Unix()       // 1779514631（线上传输的原始值）
t := inc.StartTime.Time()           // time.Time
```

请求里的时间字段仍是普通 `int64`，API 要求传数字 epoch（注意：大多数接口用**秒**，RUM 和 webhook 历史接口用**毫秒**）。

### 重试

核心包**不**内置自动重试。可以用可选的 `retry` 子包在传输层组合：它是一个默认安全的重试 `http.RoundTripper`（重试 429 和 5xx，遵循 `Retry-After`，确定性的指数退避，只重放请求体可重放的请求，SDK 的请求都满足）：

```go
import "github.com/flashcatcloud/go-flashduty/retry"

client, err := flashduty.NewClient("YOUR_APP_KEY",
	flashduty.WithTransport(retry.New(
		retry.WithMaxRetries(3),
	)),
)
```

## 开发

类型化服务层（`services_gen.go`、`models_gen.go`）由 `openapi/` 下随仓库保存的 OpenAPI 规范生成。不要手改生成文件；要改就改规范或 `internal/cmd/gen` 里的生成器。

```bash
make sync-spec   # 从 flashduty-docs 刷新 openapi/
make generate    # 重新生成类型化服务层
make check       # fmt、lint、test、build
make e2e         # 线上端到端测试；需要 FLASHDUTY_E2E_APP_KEY（可选 FLASHDUTY_E2E_BASE_URL）
```

端到端测试创建的资源都带 `gofd-e2e-` 前缀，结束时删除，不会修改已有数据。

## 相关项目

- [Flashduty CLI](https://github.com/flashcatcloud/flashduty-cli)：基于本 SDK 的 `flashduty` 命令行工具，在终端里管理故障、告警、值班和状态页（[文档](https://docs.flashduty.com/zh/developer/cli)）。
- [flashduty-mcp-server](https://github.com/flashcatcloud/flashduty-mcp-server)：让 AI 助手接入 Flashduty 的 MCP 服务。
- [terraform-provider-flashduty](https://github.com/flashcatcloud/terraform-provider-flashduty)：用代码管理 Flashduty 资源。

## 许可证

[Apache-2.0](./LICENSE)
