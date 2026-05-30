# go-flashduty

The official Go client for the [Flashduty](https://flashcat.cloud) Open API — a thin, typed SDK covering every Flashduty REST endpoint.

📖 **API reference:** <https://docs.flashcat.cloud/en/openapi/introduction>

> **Status:** All 224 Open API endpoints across 21 services are generated from the Flashduty OpenAPI specification, covered by unit tests, and validated end-to-end against the live API.

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

## Design

- **Thin and typed.** Every method maps to exactly one HTTP call and returns `(*T, *Response, error)`. No hidden cross-endpoint enrichment.
- **Service-grouped.** Endpoints are organized into services on the client (`client.Incidents`, `client.Alerts`, …), generated from the OpenAPI specification.
- **Composable transport.** Cross-cutting concerns (retry, caching, tracing, rate-limit handling) compose as `http.RoundTripper` middleware via `WithTransport`.

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

## License

[Apache-2.0](./LICENSE)
