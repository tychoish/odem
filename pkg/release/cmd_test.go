package release_test

// Binary integration tests.  A single odem binary is compiled once per
// `go test` invocation (via TestMain) and then exercised from various
// working directories to cover both normal and outside-source-tree usage.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// --------------------------------------------------------------------------
// TestMain – compile the binary once for the whole package test run.
// --------------------------------------------------------------------------

var (
	testBinaryPath string
	testBinaryOnce sync.Once
	testBinaryErr  error
)

func ensureTestBinary(t *testing.T) string {
	t.Helper()
	testBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "odem-bin-test-*")
		if err != nil {
			testBinaryErr = fmt.Errorf("MkdirTemp: %w", err)
			return
		}
		// Locate module root so `go build` runs from the right directory.
		out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").CombinedOutput()
		if err != nil {
			testBinaryErr = fmt.Errorf("go list: %w\n%s", err, out)
			return
		}
		moduleRoot := strings.TrimSpace(string(out))

		testBinaryPath = filepath.Join(dir, "odem")
		cmd := exec.Command("go", "build", "-o", testBinaryPath, "./cmd/odem.go")
		cmd.Dir = moduleRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			testBinaryErr = fmt.Errorf("go build: %w\n%s", err, out)
			testBinaryPath = ""
		}
	})
	if testBinaryErr != nil {
		t.Fatalf("could not build odem binary: %v", testBinaryErr)
	}
	return testBinaryPath
}

// --------------------------------------------------------------------------
// Helper
// --------------------------------------------------------------------------

type odemResult struct {
	Stdout   string
	Stderr   string
	Combined string
	ExitCode int
}

// runFrom invokes the test binary from dir with the given args.
// If dir is empty the test's TempDir is used.
func runFrom(t *testing.T, dir string, args ...string) odemResult {
	t.Helper()
	bin := ensureTestBinary(t)
	if dir == "" {
		dir = t.TempDir()
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}
	combined := stdout.String() + stderr.String()
	return odemResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Combined: combined,
		ExitCode: exitCode,
	}
}

// emptyConfPath writes an empty yaml config file and returns its path.
// Pass it as: odem build --conf <path> <subcommand>
// This prevents tests from inheriting deploy settings (remote, instance, etc.)
// from the developer's ~/.odem.yaml.
func emptyConfPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "empty.yaml")
	if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("emptyConfPath: %v", err)
	}
	return p
}

// --------------------------------------------------------------------------
// Basic smoke tests
// --------------------------------------------------------------------------

func TestBinaryVersionCommand(t *testing.T) {
	t.Parallel()
	r := runFrom(t, "", "version")
	if r.ExitCode != 0 {
		t.Errorf("exit=%d want 0; output:\n%s", r.ExitCode, r.Combined)
	}
	if !strings.Contains(r.Combined, "release") {
		t.Errorf("version output should contain 'release'; got:\n%s", r.Combined)
	}
}

func TestBinaryRootHelp(t *testing.T) {
	t.Parallel()
	r := runFrom(t, "", "--help")
	if r.ExitCode != 0 {
		t.Errorf("exit=%d want 0; output:\n%s", r.ExitCode, r.Combined)
	}
	for _, want := range []string{"build", "setup", "version"} {
		if !strings.Contains(r.Combined, want) {
			t.Errorf("help should mention %q; got:\n%s", want, r.Combined)
		}
	}
}

func TestBinaryBuildHelp(t *testing.T) {
	t.Parallel()
	r := runFrom(t, "", "build", "--help")
	if r.ExitCode != 0 {
		t.Errorf("exit=%d want 0; output:\n%s", r.ExitCode, r.Combined)
	}
	for _, want := range []string{"deploy", "release", "all", "link", "update"} {
		if !strings.Contains(r.Combined, want) {
			t.Errorf("build help should mention %q; got:\n%s", want, r.Combined)
		}
	}
}

// --------------------------------------------------------------------------
// Upload sub-command validation
// --------------------------------------------------------------------------

func TestBinaryUploadRequiresTag(t *testing.T) {
	t.Parallel()
	r := runFrom(t, "", "build", "release", "upload")
	if r.ExitCode == 0 {
		t.Errorf("expected non-zero exit; output:\n%s", r.Combined)
	}
	// urfave/cli v3 says "Required flag 'tag' not set"
	if !strings.Contains(r.Combined, "tag") {
		t.Errorf("error should mention 'tag'; got:\n%s", r.Combined)
	}
}

func TestBinaryUploadInvalidTag(t *testing.T) {
	t.Parallel()
	r := runFrom(t, "", "build", "release", "upload", "--tag", "notaversion")
	if r.ExitCode == 0 {
		t.Errorf("expected non-zero exit; output:\n%s", r.Combined)
	}
	if !strings.Contains(r.Combined, "notaversion") {
		t.Errorf("error should echo the bad tag; got:\n%s", r.Combined)
	}
}

// TestBinaryUploadMissingBuildDir verifies that a valid tag passes semver
// validation and the binary then fails because the build directory doesn't
// exist – not with a generic panic about an empty tag.
func TestBinaryUploadMissingBuildDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // no build/ sub-directory
	r := runFrom(t, dir, "build", "release", "upload", "--tag", "v1.2.3")
	if r.ExitCode == 0 {
		t.Errorf("expected non-zero exit; output:\n%s", r.Combined)
	}
	if !strings.Contains(r.Combined, "does not exist") {
		t.Errorf("error should mention missing directory; got:\n%s", r.Combined)
	}
	// Confirm the tag reached the function (path includes tag)
	if !strings.Contains(r.Combined, "v1.2.3") {
		t.Errorf("error should include the tag in the path; got:\n%s", r.Combined)
	}
}

func TestBinaryUploadOdemPrefixedTag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r := runFrom(t, dir, "build", "release", "upload", "--tag", "odem-v2.0.0")
	if r.ExitCode == 0 {
		t.Errorf("expected non-zero exit (missing build dir); output:\n%s", r.Combined)
	}
	// Prefix-stripped or full tag should appear in the path error
	if !strings.Contains(r.Combined, "does not exist") {
		t.Errorf("error should mention missing directory; got:\n%s", r.Combined)
	}
}

// --------------------------------------------------------------------------
// Deploy sub-command validation
// --------------------------------------------------------------------------

func TestBinaryDeployRestartRequiresInstance(t *testing.T) {
	t.Parallel()
	conf := emptyConfPath(t)
	r := runFrom(t, t.TempDir(), "build", "deploy", "--conf", conf, "restart")
	if r.ExitCode == 0 {
		t.Errorf("expected non-zero exit; output:\n%s", r.Combined)
	}
	if !strings.Contains(r.Combined, "instance") {
		t.Errorf("error should mention 'instance'; got:\n%s", r.Combined)
	}
}

func TestBinaryDeployServiceRequiresInstance(t *testing.T) {
	t.Parallel()
	conf := emptyConfPath(t)
	r := runFrom(t, t.TempDir(), "build", "deploy", "--conf", conf, "service")
	if r.ExitCode == 0 {
		t.Errorf("expected non-zero exit; output:\n%s", r.Combined)
	}
	if !strings.Contains(r.Combined, "instance") {
		t.Errorf("error should mention 'instance'; got:\n%s", r.Combined)
	}
}

// --------------------------------------------------------------------------
// Outside-source-tree: LocalBuild must work from any working directory
// because basePathCandidates includes ~/src/odem/.
// --------------------------------------------------------------------------

// TestBinaryBuildFromTempDir verifies that `odem build` launched from a
// temporary directory finds the source tree via the configured fallback path.
// We only assert that the binary does NOT fail with "no odem checkout
// discoverable"; an actual compilation error (missing toolchain, etc.) is
// acceptable for a smoke test.
func TestBinaryBuildFromTempDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r := runFrom(t, dir, "build", "odem")
	if strings.Contains(r.Combined, "no odem checkout discoverable") {
		t.Errorf("binary should find source tree from temp dir; output:\n%s", r.Combined)
	}
}
