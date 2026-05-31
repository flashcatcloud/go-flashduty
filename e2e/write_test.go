//go:build e2e

package e2e

import (
	"context"
	"testing"

	flashduty "github.com/flashcatcloud/go-flashduty"
)

// TestTeamLifecycle creates a team, reads it, updates it, and deletes it. It
// only ever touches the resource it creates (a "gofd-e2e-" team), and always
// deletes it on cleanup.
func TestTeamLifecycle(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	name := uniqueName("team")
	created, _, err := c.Teams.WriteUpsert(ctx, &flashduty.TeamUpsertRequest{
		TeamName:    name,
		Description: "go-flashduty e2e test team",
	})
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	id := created.TeamID
	if id == 0 {
		t.Fatalf("create team returned no team_id")
	}
	t.Cleanup(func() {
		if _, err := c.Teams.WriteDelete(ctx, &flashduty.TeamDeleteRequest{TeamID: id}); err != nil {
			t.Errorf("cleanup: delete team %d: %v", id, err)
		}
	})

	got, _, err := c.Teams.ReadInfo(ctx, &flashduty.TeamInfoRequest{TeamID: id})
	if err != nil {
		t.Fatalf("read team: %v", err)
	}
	if got.TeamName != name {
		t.Errorf("team name = %q, want %q", got.TeamName, name)
	}

	const updatedDesc = "go-flashduty e2e test team (updated)"
	if _, _, err := c.Teams.WriteUpsert(ctx, &flashduty.TeamUpsertRequest{
		TeamID:      id,
		TeamName:    name,
		Description: updatedDesc,
	}); err != nil {
		t.Fatalf("update team: %v", err)
	}

	updated, _, err := c.Teams.ReadInfo(ctx, &flashduty.TeamInfoRequest{TeamID: id})
	if err != nil {
		t.Fatalf("read updated team: %v", err)
	}
	if updated.Description != updatedDesc {
		t.Errorf("description = %q, want %q", updated.Description, updatedDesc)
	}
}

// TestCalendarLifecycle creates a calendar, reads it, updates it, and deletes
// it — a second write-path resource (beyond teams) proving create/update/delete
// generalize across services. It only ever touches the "gofd-e2e-" calendar it
// creates and always deletes it on cleanup.
func TestCalendarLifecycle(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	name := uniqueName("cal")
	created, _, err := c.Calendars.CalendarCreate(ctx, &flashduty.CalendarCreateRequest{
		CalName:  name,
		Timezone: "Asia/Shanghai",
		Workdays: []int64{1, 2, 3, 4, 5},
	})
	if err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	id := created.CalID
	if id == "" {
		t.Fatalf("create calendar returned no cal_id")
	}
	t.Cleanup(func() {
		if _, err := c.Calendars.CalendarDelete(ctx, &flashduty.CalendarIDRequest{CalID: id}); err != nil {
			t.Errorf("cleanup: delete calendar %s: %v", id, err)
		}
	})

	got, _, err := c.Calendars.CalendarInfo(ctx, &flashduty.CalendarIDRequest{CalID: id})
	if err != nil {
		t.Fatalf("read calendar: %v", err)
	}
	if got.CalName != name {
		t.Errorf("calendar name = %q, want %q", got.CalName, name)
	}

	const updatedDesc = "go-flashduty e2e test calendar (updated)"
	if _, err := c.Calendars.CalendarUpdate(ctx, &flashduty.CalendarUpdateRequest{
		CalID:       id,
		CalName:     flashduty.String(name),
		Timezone:    flashduty.String("Asia/Shanghai"),
		Description: flashduty.String(updatedDesc),
	}); err != nil {
		t.Fatalf("update calendar: %v", err)
	}

	updated, _, err := c.Calendars.CalendarInfo(ctx, &flashduty.CalendarIDRequest{CalID: id})
	if err != nil {
		t.Fatalf("read updated calendar: %v", err)
	}
	if updated.Description != updatedDesc {
		t.Errorf("description = %q, want %q", updated.Description, updatedDesc)
	}
}
