package model

import (
	"testing"
	"time"
)

func TestValidTypesContainsFourTypes(t *testing.T) {
	expected := []string{
		TypeBrandPublish,
		TypePolicyApply,
		TypeRiskRecovery,
		TypeHighRiskAction,
	}

	if len(ValidTypes) != 4 {
		t.Fatalf("expected 4 valid types, got %d", len(ValidTypes))
	}

	for _, typ := range expected {
		if !ValidTypes[typ] {
			t.Errorf("expected ValidTypes to contain %q", typ)
		}
	}
}

func TestTerminalStatusesContainsFourStatuses(t *testing.T) {
	expected := []string{
		StatusExecuted,
		StatusRejected,
		StatusExpired,
		StatusFailed,
	}

	if len(TerminalStatuses) != 4 {
		t.Fatalf("expected 4 terminal statuses, got %d", len(TerminalStatuses))
	}

	for _, s := range expected {
		if !TerminalStatuses[s] {
			t.Errorf("expected TerminalStatuses to contain %q", s)
		}
	}
}

func TestNonTerminalStatusesExcluded(t *testing.T) {
	nonTerminal := []string{StatusPending, StatusApproved}
	for _, s := range nonTerminal {
		if TerminalStatuses[s] {
			t.Errorf("%q should not be a terminal status", s)
		}
	}
}

func TestReviewerRolesContainsPlatformAndModerator(t *testing.T) {
	if len(ReviewerRoles) != 2 {
		t.Fatalf("expected 2 reviewer roles, got %d", len(ReviewerRoles))
	}

	if !ReviewerRoles["Platform"] {
		t.Error("expected ReviewerRoles to contain Platform")
	}
	if !ReviewerRoles["Moderator"] {
		t.Error("expected ReviewerRoles to contain Moderator")
	}
}

func TestReviewerRolesExcludesOtherRoles(t *testing.T) {
	others := []string{"Brand", "Factory", "Consumer", "Admin", ""}
	for _, role := range others {
		if ReviewerRoles[role] {
			t.Errorf("%q should not be a reviewer role", role)
		}
	}
}

func TestIsExpired_ExpiredTime(t *testing.T) {
	a := &Approval{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if !a.IsExpired() {
		t.Error("expected approval with past ExpiresAt to be expired")
	}
}

func TestIsExpired_NotExpiredTime(t *testing.T) {
	a := &Approval{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if a.IsExpired() {
		t.Error("expected approval with future ExpiresAt to not be expired")
	}
}

func TestIsExpired_FarPast(t *testing.T) {
	a := &Approval{
		ExpiresAt: time.Now().Add(-72 * time.Hour),
	}
	if !a.IsExpired() {
		t.Error("expected approval expired 72h ago to be expired")
	}
}

func TestIsExpired_FarFuture(t *testing.T) {
	a := &Approval{
		ExpiresAt: time.Now().Add(72 * time.Hour),
	}
	if a.IsExpired() {
		t.Error("expected approval expiring in 72h to not be expired")
	}
}
