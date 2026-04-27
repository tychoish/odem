package release

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tychoish/odem"
)

func TestIsPrerelease(t *testing.T) {
	t.Parallel()

	cases := []struct {
		version    string
		prerelease bool
	}{
		{"v1.0.0", false},
		{"1.2.3", false},
		{"v1.0.0-alpha", true},
		{"v1.0.0-rc.1", true},
		{"v0.0.1-pre", true},
	}
	for _, tc := range cases {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()
			got := IsPrerelease(tc.version)
			if got != tc.prerelease {
				t.Errorf("IsPrerelease(%q)=%v, want %v", tc.version, got, tc.prerelease)
			}
		})
	}
}

func TestGitDescribeSmokeTest(t *testing.T) {
	// GitDescribe runs `git describe` in the current directory.
	// In the repo this should return a non-empty string.
	got := GitDescribe()
	if got == "" {
		t.Error("GitDescribe returned empty string")
	}
}

func TestUploadArtifactsMissingDir(t *testing.T) {
	t.Parallel()

	conf := &odem.Configuration{}
	conf.Build.Tag = "v1.0.0"
	conf.Build.Path = filepath.Join(t.TempDir(), "nonexistent")

	err := UploadArtifacts(context.Background(), conf)
	if err == nil {
		t.Error("expected error for missing build directory, got nil")
	}
}

func TestUploadArtifactsEmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tag := "v1.2.3"

	conf := &odem.Configuration{}
	conf.Build.Tag = tag
	conf.Build.Path = dir

	// Create the build sub-directory so UploadArtifacts proceeds past the
	// existence check, but leave it empty so no artifacts are found.
	buildDir := filepath.Join(dir, tag)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Expect no error: an empty dir is handled gracefully (warning only).
	if err := UploadArtifacts(context.Background(), conf); err != nil {
		t.Errorf("UploadArtifacts with empty dir: unexpected error: %v", err)
	}
}

func TestUploadArtifactsTagStripping(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Tag with "odem-" prefix — the function should strip the prefix for the
	// release ID but still look up the directory by the full tag.
	tag := "odem-v1.2.3"

	conf := &odem.Configuration{}
	conf.Build.Tag = tag
	conf.Build.Path = dir

	buildDir := filepath.Join(dir, tag)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Empty dir → no artifacts → no gh invocation, no error.
	if err := UploadArtifacts(context.Background(), conf); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
