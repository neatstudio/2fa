package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunAddListEditDeleteWithStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := Run([]string{"--store", path, "add", "--name", "github", "--secret", "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", "--group", "work", "--note", "admin"}, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("add returned error: %v; stderr=%s", err, errOut.String())
	}
	if strings.Contains(out.String(), "GEZDGNBVGY3TQOJQ") {
		t.Fatalf("add output leaked secret: %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	err = Run([]string{"--store", path, "--once"}, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("list returned error: %v; stderr=%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "github") || !strings.Contains(out.String(), "admin") {
		t.Fatalf("list output missing account:\n%s", out.String())
	}
	if strings.Contains(out.String(), "GEZDGNBVGY3TQOJQ") {
		t.Fatalf("list output leaked secret: %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	err = Run([]string{"--store", path, "edit", "github", "--group", "game", "--note", "steam"}, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("edit returned error: %v; stderr=%s", err, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	err = Run([]string{"--store", path, "game", "--once"}, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("group list returned error: %v; stderr=%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "github") || !strings.Contains(out.String(), "steam") || strings.Contains(out.String(), "work") {
		t.Fatalf("group list output wrong:\n%s", out.String())
	}

	out.Reset()
	errOut.Reset()
	err = Run([]string{"--store", path, "delete", "github", "--yes"}, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("delete returned error: %v; stderr=%s", err, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	err = Run([]string{"--store", path, "--once"}, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("list after delete returned error: %v; stderr=%s", err, errOut.String())
	}
	if strings.Contains(out.String(), "github") {
		t.Fatalf("deleted account still listed:\n%s", out.String())
	}
}

func TestRunRejectsDuplicateAdd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	args := []string{"--store", path, "add", "--name", "github", "--secret", "GEZDGNBVGY3TQOJQ"}
	if err := Run(args, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("first add returned error: %v", err)
	}
	if err := Run(args, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("second add succeeded, want duplicate-name error")
	}
}

func TestRunWithDevNullOutputPrintsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	done := make(chan error, 1)
	go func() {
		done <- Run([]string{"--store", path}, strings.NewReader(""), devNull, &bytes.Buffer{})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run() did not return; non-TTY character device output may be treated as an interactive terminal")
	}
}
