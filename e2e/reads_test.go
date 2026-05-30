//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	flashduty "github.com/flashcatcloud/go-flashduty"
)

// TestReads exercises a broad set of read endpoints against the live API,
// confirming requests encode and real responses decode into the SDK types.
func TestReads(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	t.Run("MemberInfo", func(t *testing.T) {
		mi, resp, err := c.Members.MemberInfo(ctx)
		if err != nil {
			t.Fatalf("MemberInfo: %v", err)
		}
		if mi.AccountID == 0 {
			t.Errorf("expected a non-zero account id")
		}
		t.Logf("account=%q request_id=%s", mi.AccountName, resp.RequestID)
	})

	t.Run("MemberList", func(t *testing.T) {
		ml, resp, err := c.Members.MemberList(ctx, &flashduty.MemberListRequest{})
		if err != nil {
			t.Fatalf("MemberList: %v", err)
		}
		t.Logf("members=%d total=%d", len(ml.Items), resp.Total)
	})

	t.Run("ChannelList", func(t *testing.T) {
		cl, resp, err := c.Channels.ChannelList(ctx, &flashduty.ListChannelsRequest{})
		if err != nil {
			t.Fatalf("ChannelList: %v", err)
		}
		t.Logf("channels=%d total=%d", len(cl.Items), resp.Total)
	})

	t.Run("TeamList", func(t *testing.T) {
		tl, _, err := c.Teams.ReadList(ctx, &flashduty.TeamListRequest{})
		if err != nil {
			t.Fatalf("Teams.ReadList: %v", err)
		}
		t.Logf("teams=%d", len(tl.Items))
	})

	t.Run("IncidentList", func(t *testing.T) {
		now := time.Now().Unix()
		il, resp, err := c.Incidents.List(ctx, &flashduty.ListIncidentsRequest{
			StartTime:   now - 30*24*3600,
			EndTime:     now,
			ListOptions: flashduty.ListOptions{Limit: 5},
		})
		if err != nil {
			t.Fatalf("Incidents.List: %v", err)
		}
		t.Logf("incidents=%d total=%d has_next=%t", len(il.Items), resp.Total, resp.HasNextPage)
	})
}

// TestPagination follows the search-after cursor across a page boundary.
func TestPagination(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()
	now := time.Now().Unix()

	req := &flashduty.ListIncidentsRequest{
		// The API caps the incident query window at < 31 days.
		StartTime:   now - 30*24*3600,
		EndTime:     now,
		ListOptions: flashduty.ListOptions{Limit: 2},
	}
	first, resp, err := c.Incidents.List(ctx, req)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if !resp.HasNextPage {
		t.Skipf("not enough incidents to paginate (total=%d)", resp.Total)
	}

	req.ListOptions.SearchAfterCtx = first.SearchAfterCtx
	second, _, err := c.Incidents.List(ctx, req)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(first.Items) > 0 && len(second.Items) > 0 && first.Items[0].IncidentID == second.Items[0].IncidentID {
		t.Errorf("cursor did not advance: page 2 starts with the same incident as page 1")
	}
	t.Logf("page1=%d page2=%d", len(first.Items), len(second.Items))
}
