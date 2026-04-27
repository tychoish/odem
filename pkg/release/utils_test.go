package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateVersion(t *testing.T) {
	t.Parallel()

	valid := []string{
		"v1.0.0",
		"1.2.3",
		"v0.0.1",
		"v1.2.3-alpha",
		"v1.2.3-rc.1",
		"odem-v1.0.0",
		"odem-1.2.3",
		"odem-v1.2.3-alpha",
	}
	for _, tag := range valid {
		t.Run(tag, func(t *testing.T) {
			t.Parallel()
			if err := ValidateVersion(tag); err != nil {
				t.Errorf("ValidateVersion(%q) unexpected error: %v", tag, err)
			}
		})
	}

	invalid := []string{
		"not-a-version",
		"abc",
		"totally-wrong",
	}
	for _, tag := range invalid {
		t.Run("invalid/"+tag, func(t *testing.T) {
			t.Parallel()
			if err := ValidateVersion(tag); err == nil {
				t.Errorf("ValidateVersion(%q) expected error, got nil", tag)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	existing := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(existing, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !fileExists(existing) {
		t.Error("expected fileExists to return true for existing file")
	}
	if fileExists(filepath.Join(dir, "missing.txt")) {
		t.Error("expected fileExists to return false for missing file")
	}
	if !fileExists(dir) {
		t.Error("expected fileExists to return true for existing directory")
	}
}

func TestMkdirdashp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	newDir := filepath.Join(dir, "a", "b", "c")

	if err := mkdirdashp(newDir); err != nil {
		t.Fatalf("mkdirdashp: %v", err)
	}
	if !fileExists(newDir) {
		t.Error("directory was not created")
	}

	// calling again on existing dir should be a no-op
	if err := mkdirdashp(newDir); err != nil {
		t.Fatalf("mkdirdashp on existing dir: %v", err)
	}
}

func TestJoinHelpers(t *testing.T) {
	t.Parallel()

	if got := joinstr("a", "b", "c"); got != "abc" {
		t.Errorf("joinstr: got %q, want %q", got, "abc")
	}
	if got := joindot("a", "b", "c"); got != "a.b.c" {
		t.Errorf("joindot: got %q, want %q", got, "a.b.c")
	}
	if got := joindash("a", "b", "c"); got != "a-b-c" {
		t.Errorf("joindash: got %q, want %q", got, "a-b-c")
	}
}