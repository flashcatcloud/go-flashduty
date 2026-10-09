# go-flashduty

[![Go Reference](https://pkg.go.dev/badge/github.com/flashcatcloud/go-flashduty.svg)](https://pkg.go.dev/github.com/flashcatcloud/go-flashduty)
[![CI](https://img.shields.io/github/actions/workflow/status/flashcatcloud/go-flashduty/ci.yml?style=flat-square&branch=main&label=CI)](https://github.com/flashcatcloud/go-flashduty/actions)
[![Release](https://img.shields.io/github/v/tag/flashcatcloud/go-flashduty?style=flat-square&color=24bfa5&label=release)](https://github.com/flashcatcloud/go-flashduty/tags)
[![License](https://img.shields.io/github/license/flashcatcloud/go-flashduty?style=flat-square&color=24bfa5)](LICENSE)

**go-flashduty** is the official Go SDK for the [Flashduty](https://www.flashduty.com) Open API. Flashduty is an incident management and on-call platform; this client gives Go programs typed access to every public endpoint: incidents, alerts, channels, on-call schedules, status pages, monitors, RUM, and the AI SRE.

[Website](https://www.flashduty.com) · [API reference](https://docs.flashduty.com/en/openapi/introduction) · [Go package docs](https://pkg.go.dev/github.com/flashcatcloud/go-flashduty) · [Console](https://console.flashcat.cloud) · [Flashduty CLI](https://github.com/flashcatcloud/flashduty-cli)

> **Status:** Typed API operations are generated from the Flashduty OpenAPI specification, covered by unit tests, and validated end-to-end against the live API.

## Install

```bash
go get github.com/flashcatcloud/go-flashduty
```

Requires Go 1.24+.

## Quick start

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

## Authentication

Create an APP key in the [console](https://console.flashcat.cloud) (My → APP Key); the [API reference](https://docs.flashduty.com/en/openapi/introduction) describes the steps. `NewClient` sends it as the `app_key` query parameter.

To call the API with an OAuth access token instead, use `NewClientWithAccessToken`; it sends `Authorization: Bearer <token>` and accepts the same options:

```go
client, err := flashduty.NewClientWithAccessToken(accessToken)
```

## Design

- **Thin and typed.** Every method maps to exactly one HTTP call and returns `(*T, *Response, error)`. No hidden cross-endpoint enrichment.
- **Service-grouped.** Endpoints are organized into services on the client (`client.Incidents`, `client.Alerts`, …), generated from the OpenAPI specification.
- **Composable transport.** Cross-cutting concerns (retry, caching, tracing, rate-limit handling) compose as `http.RoundTripper` middleware via `WithTransport`.
- **Human-readable timestamps.** Response time fields are typed `Timestamp` (Unix seconds) or `TimestampMilli` (milliseconds) instead of bare integers, so JSON, logs, and LLM-facing output read as RFC3339 — while the raw epoch is one call away. Request fields stay plain `int64`.

### Options

```go
client, err := flashduty.NewClient("YOUR_APP_KEY",
	flashduty.WithBaseURL("https://api.flashcat.cloud"),
	flashduty.WithTimeout(10*time.Second),
	flashduty.WithUserAgent("my-app/1.0"),
	flashduty.WithHTTPClient(customHTTPClient),
	flashduty.WithTransport(customRoundTripper),
	flashduty.WithLogger(myLogger),
	flashduty.WithRequestHeaders(staticHeaders),
	flashduty.WithRequestHook(func(req *http.Request) { /* e.g. inject traceparent */ }),
)
```

### Pagination

List requests embed `ListOptions`. Zero values are omitted, so the server defaults apply (page 1, 20 items). `Response.Total` and `Response.HasNextPage` describe the result set:

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

Endpoints that support deep pagination return an opaque cursor in `Response.SearchAfterCtx`; pass it back in `ListOptions.SearchAfterCtx` to fetch the next page.

### Errors and rate limits

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

Typed predicates save you the string comparison and see through wrapped errors
(`errors.As` under the hood):

```go
if flashduty.IsNotFound(err) { /* ... */ }
if flashduty.IsRateLimited(err) { /* ... */ }
switch flashduty.ErrorCodeOf(err) {
case flashduty.ErrorCodeAccessDenied, flashduty.ErrorCodeUnauthorized:
	// handle auth failures
}
```

### Timestamps

Time fields on responses are `Timestamp` (Unix seconds) or `TimestampMilli`
(milliseconds). They marshal to an RFC3339 string in the local timezone and
unmarshal from either a numeric epoch or an RFC3339 string, so a value round-trips
cleanly. The zero value stays the numeric `0` sentinel (never a 1970 date) and is
dropped by `omitempty`.

```go
inc := list.Items[0]
fmt.Println(inc.StartTime)          // 2026-05-30T14:37:11+08:00  (String / fmt / TOON)
b, _ := json.Marshal(inc.StartTime) // "2026-05-30T14:37:11+08:00"
epoch := inc.StartTime.Unix()       // 1779514631  (raw wire value)
t := inc.StartTime.Time()           // time.Time
```

Request time fields stay plain `int64` — the API expects a numeric epoch on the
wire (note: most endpoints take **seconds**, but RUM and webhook-history
endpoints take **milliseconds**).

### Retries

Automatic retries are **not** built into the core. Compose them at the transport
layer with the optional `retry` subpackage — a safe-by-default retrying
`http.RoundTripper` (retries 429 and 5xx, honors `Retry-After`, deterministic
exponential backoff, and only replays requests whose body is replayable, which
all SDK requests are):

```go
import "github.com/flashcatcloud/go-flashduty/retry"

client, err := flashduty.NewClient("YOUR_APP_KEY",
	flashduty.WithTransport(retry.New(
		retry.WithMaxRetries(3),
	)),
)
```

## Development

The typed service layer (`services_gen.go`, `models_gen.go`) is generated from the OpenAPI spec vendored in `openapi/`. Don't edit the generated files; change the spec or the generator in `internal/cmd/gen`.

```bash
make sync-spec   # refresh openapi/ from flashduty-docs
make generate    # regenerate the typed service layer
make check       # fmt, lint, test, build
make e2e         # live end-to-end tests; needs FLASHDUTY_E2E_APP_KEY (optional FLASHDUTY_E2E_BASE_URL)
```

End-to-end tests create resources with a `gofd-e2e-` prefix and delete them on cleanup; they never modify existing data.

## Related projects

- [Flashduty CLI](https://github.com/flashcatcloud/flashduty-cli) — the `flashduty` command-line tool, built on this SDK, for managing incidents, alerts, on-call and status pages from a terminal ([docs](https://docs.flashduty.com/en/developer/cli)).
- [flashduty-mcp-server](https://github.com/flashcatcloud/flashduty-mcp-server) — MCP server exposing Flashduty to AI assistants.
- [terraform-provider-flashduty](https://github.com/flashcatcloud/terraform-provider-flashduty) — manage Flashduty resources as code.

## License

[Apache-2.0](./LICENSE)
