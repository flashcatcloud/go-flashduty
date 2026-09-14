package flashduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ReadPrometheusLabelValues lists the values of one label from a
// Prometheus-compatible data source, through the Monitors proxy.
//
// Unlike the generated typed endpoints, the response body is the data
// source's native Prometheus HTTP API payload: it is not wrapped in the
// Flashduty {request_id, error, data} envelope, so decoding goes straight
// into PrometheusLabelValuesResponse. A non-2xx status can come from either
// side of the proxy — the platform itself (plain text, raised before the
// data source is reached) or the data source (its own JSON error shape,
// distinct from Flashduty's) — so the raw body is carried as-is on the
// returned error's Message rather than parsed as a Flashduty error.
//
// API: GET /monit/prometheus/api/v1/label/{label_name}/values (monit-prometheus-read-label-values).
func (s *DataSourcesService) ReadPrometheusLabelValues(ctx context.Context, dataSourceID uint64, labelName string) (*PrometheusLabelValuesResponse, *Response, error) {
	if dataSourceID == 0 {
		return nil, nil, fmt.Errorf("flashduty: data source id is required")
	}
	labelName = strings.TrimSpace(labelName)
	if labelName == "" {
		return nil, nil, fmt.Errorf("flashduty: label name is required")
	}

	path := "/monit/prometheus/api/v1/label/" + url.PathEscape(labelName) + "/values"
	httpReq, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("X-DSID", strconv.FormatUint(dataSourceID, 10))

	httpResp, err := s.client.client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("flashduty: request to %s failed: %v", sanitizeURL(httpReq.URL), sanitizeError(err))
	}
	defer func() { _ = httpResp.Body.Close() }()

	resp := &Response{Response: httpResp, RequestID: httpResp.Header.Get("Flashcat-Request-Id")}
	resp.RateLimit = parseRateLimit(httpResp.Header)

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBodySize))
	if err != nil {
		return nil, resp, fmt.Errorf("flashduty: reading response body: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		apiErr := &ErrorResponse{Response: httpResp, Message: strings.TrimSpace(string(raw)), RequestID: resp.RequestID}
		return nil, resp, asAPIError(apiErr, resp.RateLimit)
	}

	out := new(PrometheusLabelValuesResponse)
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return nil, resp, fmt.Errorf("flashduty: decoding response into %T (request_id %s): %w", out, resp.RequestID, err)
		}
	}
	return out, resp, nil
}
