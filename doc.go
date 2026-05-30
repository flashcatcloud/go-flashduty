// Package flashduty is the official Go client for the Flashduty Open API
// (https://flashcat.cloud). It is a thin, typed wrapper: every method maps to
// exactly one HTTP call, returns (*T, *Response, error), and performs no hidden
// cross-endpoint enrichment.
//
// Create a client with an app key:
//
//	client, err := flashduty.NewClient("APP_KEY")
//	if err != nil {
//		// handle error
//	}
//
// All endpoints are POST actions grouped into services on the client, e.g.
// client.Incidents.List(ctx, &flashduty.IncidentListRequest{...}). Services are
// added by the code generator; see internal/cmd/gen.
//
// Cross-cutting transport concerns (retry, caching, tracing, rate-limit
// handling) compose as http.RoundTripper middleware via WithTransport, mirroring
// google/go-github. The optional flashdutyretry subpackage provides a
// safe-by-default retrying transport.
package flashduty
