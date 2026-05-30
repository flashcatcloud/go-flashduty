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
