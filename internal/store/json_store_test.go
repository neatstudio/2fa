package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gouki/tools/2fa/internal/model"
)

func TestSaveCreatesPrivateDirAndFileAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secure", "accounts.json")
	st := New(path)
	now := time.Date(2026, 5, 20, 1, 2, 3, 0, time.UTC)
	account, err := model.NewAccount("github", "GEZDGNBVGY3TQOJQ", "work", "login", now)
	if err != nil {
		t.Fatalf("NewAccount() returned error: %v", err)
	}

	if err := st.Save(model.Accounts{account}); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("dir perm = %o, want 700", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("file perm = %o, want 600", got)
	}

	loaded, err := st.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Name != "github" || loaded[0].Secret != account.Secret {
		t.Fatalf("loaded accounts = %+v", loaded)
	}
}

func TestLoadRepairsLoosePermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"accounts":[]}`), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	st := New(path)
	if _, err := st.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("dir perm = %o, want 700", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("file perm = %o, want 600", got)
	}
}

func TestCustomStoreDoesNotChmodExistingParentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	path := filepath.Join(dir, "accounts.json")

	now := time.Date(2026, 5, 20, 1, 2, 3, 0, time.UTC)
	account, err := model.NewAccount("github", "GEZDGNBVGY3TQOJQ", "work", "login", now)
	if err != nil {
		t.Fatalf("NewAccount() returned error: %v", err)
	}

	st := New(path)
	if err := st.Save(model.Accounts{account}); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o755 {
		t.Fatalf("dir perm after Save = %o, want existing 755 unchanged", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("file perm after Save = %o, want 600", got)
	}

	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod file: %v", err)
	}
	if _, err := st.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	dirInfo, err = os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o755 {
		t.Fatalf("dir perm after Load = %o, want existing 755 unchanged", got)
	}
	fileInfo, err = os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("file perm after Load = %o, want 600", got)
	}
}
