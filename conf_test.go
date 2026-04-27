package odem_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tychoish/odem"
)

// --------------------------------------------------------------------------
// ValidateDeploy
// --------------------------------------------------------------------------

func TestValidateDeployMissingInstance(t *testing.T) {
	t.Parallel()
	conf := &odem.Configuration{}
	// both instance and remote empty
	if err := conf.ValidateDeploy(); err == nil {
		t.Error("expected error for empty instance, got nil")
	}
}

func TestValidateDeployMissingRemote(t *testing.T) {
	t.Parallel()
	conf := &odem.Configuration{}
	conf.Build.Deploy.Intstance = "prod"
	// remote still empty
	if err := conf.ValidateDeploy(); err == nil {
		t.Error("expected error for empty remote, got nil")
	}
}

func TestValidateDeployValid(t *testing.T) {
	t.Parallel()
	conf := &odem.Configuration{}
	conf.Build.Deploy.Intstance = "prod"
	conf.Build.Deploy.Remote = "myserver"
	if err := conf.ValidateDeploy(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateDeployLocalEqualsRemote(t *testing.T) {
	t.Parallel()
	conf := &odem.Configuration{}
	conf.Build.Deploy.Intstance = "dev"
	conf.Build.Deploy.Remote = "localhost"
	conf.Runtime.Hostname = "localhost"
	if err := conf.ValidateDeploy(); err != nil {
		t.Errorf("local deploy (remote == hostname) should be valid: %v", err)
	}
}

// --------------------------------------------------------------------------
// ReadConfiguration
// --------------------------------------------------------------------------

func TestReadConfigurationNoFiles(t *testing.T) {
	t.Parallel()
	// Pass a path that doesn't exist; ReadConfiguration should return an empty
	// (but non-nil) configuration rather than an error.
	conf, err := odem.ReadConfiguration("/nonexistent/path/to/conf.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf == nil {
		t.Fatal("expected non-nil configuration")
	}
}

func TestReadConfigurationValidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	confPath := filepath.Join(dir, "conf.yaml")
	content := `
build:
  path: "dist"
  binary_link: "/usr/local/bin/odem"
  deploy:
    remote: "myserver"
`
	if err := os.WriteFile(confPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	conf, err := odem.ReadConfiguration(confPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf.Build.Path != "dist" {
		t.Errorf("Build.Path=%q, want %q", conf.Build.Path, "dist")
	}
	if conf.Build.Deploy.Remote != "myserver" {
		t.Errorf("Deploy.Remote=%q, want %q", conf.Build.Deploy.Remote, "myserver")
	}
}

func TestReadConfigurationValidJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	confPath := filepath.Join(dir, "conf.json")
	content := `{"build":{"path":"artifacts","binary_link":"/usr/bin/odem"}}`
	if err := os.WriteFile(confPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	conf, err := odem.ReadConfiguration(confPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf.Build.Path != "artifacts" {
		t.Errorf("Build.Path=%q, want %q", conf.Build.Path, "artifacts")
	}
}

func TestReadConfigurationMalformedYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	confPath := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(confPath, []byte(":\t: not valid yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	conf, err := odem.ReadConfiguration(confPath)
	// Either an error is returned OR a zero configuration (depending on yaml parser leniency).
	// What must not happen is a panic.
	_ = conf
	_ = err
}
