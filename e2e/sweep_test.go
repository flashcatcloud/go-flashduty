//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	flashduty "github.com/flashcatcloud/go-flashduty"
)

// TestServiceReadSweep exercises at least one read endpoint on every one of the
// 21 generated services against the live API, proving that each service's
// request encodes, the round-trip succeeds, and the real response decodes into
// the SDK's typed model (a nil error from do()/doGet() means json.Unmarshal into
// the typed response struct succeeded).
//
// Each service is an independent subtest so one failure does not mask the rest.
// The generic *Response envelope carries request_id and the list total, so the
// sweep logs those uniformly without depending on each response's field names —
// per-field usability is covered by the README example (example_test.go), the
// incident reads, and the team write lifecycle.
func TestServiceReadSweep(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()
	now := time.Now().Unix()
	weekAgo := now - 7*24*3600
	// RUM and webhook-history endpoints take millisecond timestamps, unlike the
	// core platform (incidents/alerts/analytics) which takes seconds.
	nowMs := now * 1000
	weekAgoMs := weekAgo * 1000

	type readFn func() (requestID string, total int, err error)

	// Endpoints that require a pre-existing resource id the sweep cannot obtain
	// generically (e.g. a status-page id — this API surface has no list-pages
	// endpoint). These are skipped with the server's reason when the server
	// rejects on InvalidParameter, rather than failed — never silently passed.
	needsResource := map[string]bool{"StatusPages": true, "RuleSets": true}

	// One representative read per service. Listed in client field order.
	reads := []struct {
		service string
		run     readFn
	}{
		{"Members", func() (string, int, error) {
			_, r, err := c.Members.MemberInfo(ctx)
			return rid(r), tot(r), err
		}},
		{"Teams", func() (string, int, error) {
			_, r, err := c.Teams.ReadList(ctx, &flashduty.TeamListRequest{})
			return rid(r), tot(r), err
		}},
		{"Channels", func() (string, int, error) {
			_, r, err := c.Channels.ChannelList(ctx, &flashduty.ListChannelsRequest{})
			return rid(r), tot(r), err
		}},
		{"Incidents", func() (string, int, error) {
			_, r, err := c.Incidents.List(ctx, &flashduty.ListIncidentsRequest{
				StartTime: weekAgo, EndTime: now, ListOptions: flashduty.ListOptions{Limit: 3},
			})
			return rid(r), tot(r), err
		}},
		{"Alerts", func() (string, int, error) {
			_, r, err := c.Alerts.ReadList(ctx, &flashduty.AlertListRequest{
				StartTime: weekAgo, EndTime: now, ListOptions: flashduty.ListOptions{Limit: 3},
			})
			return rid(r), tot(r), err
		}},
		{"AlertRules", func() (string, int, error) {
			// ReadList needs a folder context; this param-less counter read also
			// exercises the map[string]int64 response type fixed in v0.2.0.
			_, r, err := c.AlertRules.ReadCounterChannel(ctx)
			return rid(r), tot(r), err
		}},
		{"AlertEnrichment", func() (string, int, error) {
			// EnrichmentReadList requires integration ids; FieldReadList is the
			// account-level, parameter-free read for this service.
			_, r, err := c.AlertEnrichment.FieldReadList(ctx, &flashduty.FieldListRequest{})
			return rid(r), tot(r), err
		}},
		{"DataSources", func() (string, int, error) {
			_, r, err := c.DataSources.ReadList(ctx, &flashduty.DataSourceListRequest{})
			return rid(r), tot(r), err
		}},
		{"Integrations", func() (string, int, error) {
			_, r, err := c.Integrations.List(ctx, &flashduty.ListWebhookHistoryRequest{
				StartTime: weekAgoMs, EndTime: nowMs, Limit: 3,
			})
			return rid(r), tot(r), err
		}},
		{"NotificationTemplates", func() (string, int, error) {
			_, r, err := c.NotificationTemplates.ReadList(ctx, &flashduty.TemplateListRequest{})
			return rid(r), tot(r), err
		}},
		{"RolesPermissions", func() (string, int, error) {
			_, r, err := c.RolesPermissions.ReadList(ctx, &flashduty.RoleListRequest{})
			return rid(r), tot(r), err
		}},
		{"RuleSets", func() (string, int, error) {
			_, r, err := c.RuleSets.List(ctx, &flashduty.StoreRulesetListRequest{TypeIdent: "prometheus"})
			return rid(r), tot(r), err
		}},
		{"Schedules", func() (string, int, error) {
			_, r, err := c.Schedules.List(ctx, &flashduty.ScheduleListRequest{})
			return rid(r), tot(r), err
		}},
		{"Calendars", func() (string, int, error) {
			_, r, err := c.Calendars.CalendarList(ctx, &flashduty.CalendarListRequest{})
			return rid(r), tot(r), err
		}},
		{"AuditLogs", func() (string, int, error) {
			_, r, err := c.AuditLogs.OperationList(ctx)
			return rid(r), tot(r), err
		}},
		{"Analytics", func() (string, int, error) {
			_, r, err := c.Analytics.IncidentList(ctx, &flashduty.InsightIncidentListRequest{
				StartTime: weekAgo, EndTime: now,
			})
			return rid(r), tot(r), err
		}},
		{"StatusPages", func() (string, int, error) {
			_, r, err := c.StatusPages.ChangeList(ctx, &flashduty.StatusPagesChangeListRequest{})
			return rid(r), tot(r), err
		}},
		{"Applications", func() (string, int, error) {
			_, r, err := c.Applications.ReadList(ctx, &flashduty.RUMApplicationListRequest{})
			return rid(r), tot(r), err
		}},
		{"Issues", func() (string, int, error) {
			_, r, err := c.Issues.ReadList(ctx, &flashduty.RUMIssueListRequest{
				StartTime: weekAgoMs, EndTime: nowMs,
			})
			return rid(r), tot(r), err
		}},
		{"Sourcemaps", func() (string, int, error) {
			_, r, err := c.Sourcemaps.List(ctx, &flashduty.SourcemapListRequest{
				StartTime: weekAgoMs, EndTime: nowMs,
			})
			return rid(r), tot(r), err
		}},
		{"Diagnostics", func() (string, int, error) {
			_, r, err := c.Diagnostics.TargetsList(ctx, &flashduty.TargetsListRequest{})
			return rid(r), tot(r), err
		}},
	}

	for _, rd := range reads {
		t.Run(rd.service, func(t *testing.T) {
			requestID, total, err := rd.run()
			if err != nil {
				if needsResource[rd.service] && flashduty.IsInvalidParameter(err) {
					t.Skipf("requires deployment-specific context (a resource id or valid type identifier) the sweep cannot synthesize: %v", err)
				}
				t.Fatalf("read failed: %v", err)
			}
			t.Logf("ok request_id=%s total=%d", requestID, total)
		})
	}
}

// rid/tot read the generic envelope fields off any service method's *Response,
// tolerating a nil response (e.g. when an error occurred before the round-trip).
func rid(r *flashduty.Response) string {
	if r == nil {
		return ""
	}
	return r.RequestID
}

func tot(r *flashduty.Response) int {
	if r == nil {
		return 0
	}
	return r.Total
}
