package flashduty

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Request-side params types for the `<type>.query` datasource tools invoked
// through DataSourcesService.ToolsInvoke.
//
// These schemas are hand-written (not generated): they are not reachable from
// any requestBody in the OpenAPI spec — the invoke request carries `params` as
// raw JSON — so the generator would emit them as response-side types, where
// epoch-millis fields marshal to RFC3339 strings and optional fields lose
// `,omitempty`. Here, optional fields are pointers: a nil pointer omits the
// key so the server keeps its default, while a non-nil pointer sends the value
// explicitly (explicit null is rejected by the server, so there is no way to
// send one). Use the Bool/Int64/String helpers in ptr.go to set them.

// DatasourceQueryExecution describes when a `<type>.query` tool evaluates.
//
// Kind selects the evaluation mode:
//   - `instant` evaluates once at ToMS; only ToMS is set.
//   - `range` evaluates a stepped series over [FromMS, ToMS]; MaxDataPoints is
//     required and bounds the returned points (the server computes the step).
//   - `window` evaluates one bounded window [FromMS, ToMS].
//
// The reserved `step_seconds` field is not modeled and must stay omitted.
type DatasourceQueryExecution struct {
	Kind string `json:"kind"`
	// Window or range start as a Unix epoch timestamp in milliseconds.
	FromMS *int64 `json:"from_ms,omitempty"`
	// Query end time as a Unix epoch timestamp in milliseconds; for `instant`
	// it is the evaluation timestamp.
	ToMS *int64 `json:"to_ms,omitempty"`
	// `range` only, required: maximum returned data points.
	MaxDataPoints *int64 `json:"max_data_points,omitempty"`
	// `range` only, optional: positive lower bound in seconds for the computed
	// step.
	MinStepSeconds *int64 `json:"min_step_seconds,omitempty"`
}

// PrometheusQueryParams are the params for `prometheus.query`. Expr is PromQL.
// Execution.Kind must be `instant` or `range`; `instant` accepts only ToMS.
type PrometheusQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
}

// MySQLQueryParams are the params for `mysql.query`. Expr is a single
// read-only SQL statement. Execution.Kind must be `window`.
type MySQLQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
}

// PostgresQueryParams are the params for `postgres.query`. Expr is a single
// read-only SQL statement. Execution.Kind must be `window`.
type PostgresQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
}

// OracleQueryParams are the params for `oracle.query`. Expr is a single
// read-only SQL statement. Execution.Kind must be `window`.
type OracleQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
}

// ClickHouseQueryParams are the params for `clickhouse.query`. Expr is a
// single read-only SQL statement. Execution.Kind must be `window`.
type ClickHouseQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
}

// ElasticsearchQueryParams are the params for `elasticsearch.query`. Expr is
// a single SQL statement; Elasticsearch DSL queries are not supported.
// Execution.Kind must be `window`.
type ElasticsearchQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
}

// LokiQueryParams are the params for `loki.query`. Expr is LogQL.
// Execution.Kind must be `instant` or `range`; `instant` keeps the full time
// context so `$__auto` ranges resolve. Limit and Direction only apply to
// raw-log results.
type LokiQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
	// Maximum raw-log entries to return (1–1000); only bounds raw logs, not
	// SQL rows or scanned data.
	Limit *int64 `json:"limit,omitempty"`
	// Raw-log retrieval order: `latest` or `earliest`.
	Direction *string `json:"direction,omitempty"`
}

// VictoriaLogsQueryParams are the params for `victorialogs.query`. Expr is
// LogsQL. Use a `window` execution for raw logs (Limit/Direction allowed) or
// an `instant`/`range` execution with FromMS for stats queries
// (Limit/Direction rejected).
type VictoriaLogsQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
	// Maximum raw-log entries to return (1–1000); only bounds raw logs, not
	// SQL rows or scanned data.
	Limit *int64 `json:"limit,omitempty"`
	// Raw-log retrieval order: `latest` or `earliest`.
	Direction *string `json:"direction,omitempty"`
}

// SLSQueryParams are the params for `sls.query` on Alibaba Cloud SLS.
// Execution.Kind must be `window`.
type SLSQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
	// SLS project name.
	Project string `json:"project"`
	// SLS logstore name.
	Logstore string `json:"logstore"`
	// Whether to run the query with SLS PowerSQL. A pointer because explicit
	// false is a real value distinct from the executor default.
	PowerSQL *bool `json:"powersql,omitempty"`
	// Maximum raw-log entries to return (1–100); only bounds raw logs, not
	// SQL rows or scanned data.
	Limit *int64 `json:"limit,omitempty"`
	// Raw-log retrieval order: `latest` or `earliest`.
	Direction *string `json:"direction,omitempty"`
}

// TencentCLSQueryParams are the params for `tencent_cls.query` on Tencent
// Cloud CLS. Execution.Kind must be `window`.
type TencentCLSQueryParams struct {
	Expr      string                   `json:"expr"`
	Execution DatasourceQueryExecution `json:"execution"`
	// Tencent Cloud region, e.g. `ap-guangzhou`.
	Region string `json:"region"`
	// CLS log topic ID.
	TopicID string `json:"topic_id"`
	// Search syntax: `cql` or `lucene`.
	Syntax string `json:"syntax"`
	// Maximum raw-log entries to return (1–1000); only bounds raw logs, not
	// SQL rows or scanned data.
	Limit *int64 `json:"limit,omitempty"`
	// Raw-log retrieval order: `latest` or `earliest`.
	Direction *string `json:"direction,omitempty"`
}

// NewDatasourceQueryInvokeRequest builds a ToolsInvoke request for a
// `<type>.query` tool by marshaling params (one of the *QueryParams types
// above) into the request's raw JSON Params. A nil params returns an error:
// query tools must not omit their params.
func NewDatasourceQueryInvokeRequest(datasourceID uint64, tool string, params any) (*DatasourceToolInvokeRequest, error) {
	if params == nil {
		return nil, errors.New("flashduty: query tool params must not be nil")
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("flashduty: marshal query tool params: %w", err)
	}
	return &DatasourceToolInvokeRequest{
		DatasourceID: datasourceID,
		Tool:         tool,
		Params:       json.RawMessage(raw),
	}, nil
}
