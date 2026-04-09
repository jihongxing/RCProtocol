package model

import "testing"

func TestValidTypesContainsThreeTypes(t *testing.T) {
	expected := []string{TypeRisk, TypeDispute, TypeRecovery}

	if len(ValidTypes) != 3 {
		t.Fatalf("expected 3 valid types, got %d", len(ValidTypes))
	}

	for _, typ := range expected {
		if !ValidTypes[typ] {
			t.Errorf("expected ValidTypes to contain %q", typ)
		}
	}
}

func TestValidConclusionTypesContainsFiveTypes(t *testing.T) {
	expected := []string{
		ConclusionFreeze,
		ConclusionRecover,
		ConclusionMarkTampered,
		ConclusionMarkCompromised,
		ConclusionDismiss,
	}

	if len(ValidConclusionTypes) != 5 {
		t.Fatalf("expected 5 valid conclusion types, got %d", len(ValidConclusionTypes))
	}

	for _, ct := range expected {
		if !ValidConclusionTypes[ct] {
			t.Errorf("expected ValidConclusionTypes to contain %q", ct)
		}
	}
}

func TestTerminalStatusesContainsClosedAndCancelled(t *testing.T) {
	if len(TerminalStatuses) != 2 {
		t.Fatalf("expected 2 terminal statuses, got %d", len(TerminalStatuses))
	}

	if !TerminalStatuses[StatusClosed] {
		t.Error("expected TerminalStatuses to contain closed")
	}
	if !TerminalStatuses[StatusCancelled] {
		t.Error("expected TerminalStatuses to contain cancelled")
	}
}

func TestNonTerminalStatusesExcluded(t *testing.T) {
	nonTerminal := []string{StatusOpen, StatusAssigned, StatusInProgress, StatusResolved}
	for _, s := range nonTerminal {
		if TerminalStatuses[s] {
			t.Errorf("%q should not be a terminal status", s)
		}
	}
}

func TestManagerRolesContainsPlatformAndModerator(t *testing.T) {
	if len(ManagerRoles) != 2 {
		t.Fatalf("expected 2 manager roles, got %d", len(ManagerRoles))
	}

	if !ManagerRoles["Platform"] {
		t.Error("expected ManagerRoles to contain Platform")
	}
	if !ManagerRoles["Moderator"] {
		t.Error("expected ManagerRoles to contain Moderator")
	}
}

func TestManagerRolesExcludesOtherRoles(t *testing.T) {
	others := []string{"Brand", "Factory", "Consumer", "Admin", ""}
	for _, role := range others {
		if ManagerRoles[role] {
			t.Errorf("%q should not be a manager role", role)
		}
	}
}

func TestAdvancableStatuses(t *testing.T) {
	if len(AdvancableStatuses) != 2 {
		t.Fatalf("expected 2 advancable statuses, got %d", len(AdvancableStatuses))
	}

	if !AdvancableStatuses[StatusAssigned] {
		t.Error("expected AdvancableStatuses to contain assigned")
	}
	if !AdvancableStatuses[StatusInProgress] {
		t.Error("expected AdvancableStatuses to contain in_progress")
	}
}

func TestCancellableStatuses(t *testing.T) {
	if len(CancellableStatuses) != 2 {
		t.Fatalf("expected 2 cancellable statuses, got %d", len(CancellableStatuses))
	}

	if !CancellableStatuses[StatusOpen] {
		t.Error("expected CancellableStatuses to contain open")
	}
	if !CancellableStatuses[StatusAssigned] {
		t.Error("expected CancellableStatuses to contain assigned")
	}
}

func TestAssignableStatuses(t *testing.T) {
	if len(AssignableStatuses) != 2 {
		t.Fatalf("expected 2 assignable statuses, got %d", len(AssignableStatuses))
	}

	if !AssignableStatuses[StatusOpen] {
		t.Error("expected AssignableStatuses to contain open")
	}
	if !AssignableStatuses[StatusAssigned] {
		t.Error("expected AssignableStatuses to contain assigned")
	}
}
