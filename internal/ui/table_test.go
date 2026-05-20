package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gouki/tools/2fa/internal/model"
)

func TestRenderTableShowsCodesAndDoesNotLeakSecrets(t *testing.T) {
	now := time.Unix(59, 0).UTC()
	account, err := model.NewAccount("github", "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", "work", "admin login", now)
	if err != nil {
		t.Fatalf("NewAccount() returned error: %v", err)
	}

	table, err := RenderTable(model.Accounts{account}, now)
	if err != nil {
		t.Fatalf("RenderTable() returned error: %v", err)
	}

	for _, want := range []string{"GROUP", "NAME", "CODE", "REMAINING", "NOTE", "work", "github", "287082", "1", "admin login"} {
		if !strings.Contains(table, want) {
			t.Fatalf("table missing %q:\n%s", want, table)
		}
	}
	if strings.Contains(table, account.Secret) {
		t.Fatalf("table leaked secret:\n%s", table)
	}
}
