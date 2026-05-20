package model

import (
	"testing"
	"time"
)

func TestNewAccountNormalizesDefaultsAndTimestamps(t *testing.T) {
	now := time.Date(2026, 5, 20, 1, 2, 3, 0, time.UTC)
	account, err := NewAccount("github", " ge zdgnbv gy3tqojq ", "", "work login", now)
	if err != nil {
		t.Fatalf("NewAccount() returned error: %v", err)
	}

	if account.Name != "github" {
		t.Fatalf("name = %q", account.Name)
	}
	if account.Group != DefaultGroup {
		t.Fatalf("group = %q, want %q", account.Group, DefaultGroup)
	}
	if account.Secret != "GEZDGNBVGY3TQOJQ" {
		t.Fatalf("secret was not normalized: %q", account.Secret)
	}
	if !account.CreatedAt.Equal(now) || !account.UpdatedAt.Equal(now) {
		t.Fatalf("timestamps = %s/%s, want %s", account.CreatedAt, account.UpdatedAt, now)
	}
}

func TestAccountsEnforceUniqueNameAndSupportCRUD(t *testing.T) {
	now := time.Date(2026, 5, 20, 1, 2, 3, 0, time.UTC)
	first, err := NewAccount("github", "GEZDGNBVGY3TQOJQ", "work", "old", now)
	if err != nil {
		t.Fatalf("NewAccount(first) returned error: %v", err)
	}
	duplicate, err := NewAccount("github", "JBSWY3DPEHPK3PXP", "game", "dup", now)
	if err != nil {
		t.Fatalf("NewAccount(duplicate) returned error: %v", err)
	}

	accounts, err := (Accounts{}).Add(first)
	if err != nil {
		t.Fatalf("Add(first) returned error: %v", err)
	}
	if _, err := accounts.Add(duplicate); err == nil {
		t.Fatal("Add(duplicate) succeeded, want duplicate-name error")
	}

	group := "game"
	note := "updated"
	secret := "jbsw y3dp ehpk 3pxp"
	updated, err := accounts.Update("github", AccountChanges{Group: &group, Note: &note, Secret: &secret}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}
	got, ok := updated.Find("github")
	if !ok {
		t.Fatal("updated account not found")
	}
	if got.Group != "game" || got.Note != "updated" || got.Secret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("updated account = %+v", got)
	}
	if !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("timestamps not preserved/updated: %+v", got)
	}

	deleted, err := updated.Delete("github")
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}
	if len(deleted) != 0 {
		t.Fatalf("len after delete = %d, want 0", len(deleted))
	}
}

func TestAccountsFilterByGroupIsExact(t *testing.T) {
	now := time.Date(2026, 5, 20, 1, 2, 3, 0, time.UTC)
	game, _ := NewAccount("steam", "GEZDGNBVGY3TQOJQ", "game", "", now)
	games, _ := NewAccount("plural", "GEZDGNBVGY3TQOJQ", "games", "", now)
	work, _ := NewAccount("github", "GEZDGNBVGY3TQOJQ", "work", "", now)
	accounts := Accounts{game, games, work}

	filtered := accounts.FilterByGroup("game")
	if len(filtered) != 1 || filtered[0].Name != "steam" {
		t.Fatalf("FilterByGroup(game) = %+v", filtered)
	}
}
