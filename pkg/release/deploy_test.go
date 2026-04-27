package release

import (
	"context"
	"strings"
	"testing"

	"github.com/tychoish/odem"
)

func TestGetServiceRestartArgs(t *testing.T) {
	t.Parallel()

	t.Run("UserService", func(t *testing.T) {
		t.Parallel()
		conf := &odem.Configuration{}
		conf.Build.Deploy.Intstance = "default"
		conf.Build.Deploy.GlobalService = false

		args := getServiceRestartArgs(conf)
		want := []string{"systemctl", "--user", "restart", "odem@default.service"}
		if len(args) != len(want) {
			t.Fatalf("len(args)=%d, want %d; got %v", len(args), len(want), args)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Errorf("args[%d]=%q, want %q", i, args[i], want[i])
			}
		}
	})

	t.Run("GlobalService", func(t *testing.T) {
		t.Parallel()
		conf := &odem.Configuration{}
		conf.Build.Deploy.Intstance = "prod"
		conf.Build.Deploy.GlobalService = true

		args := getServiceRestartArgs(conf)
		want := []string{"sudo", "systemctl", "restart", "odem@prod.service"}
		if len(args) != len(want) {
			t.Fatalf("len(args)=%d, want %d; got %v", len(args), len(want), args)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Errorf("args[%d]=%q, want %q", i, args[i], want[i])
			}
		}
	})

	t.Run("EmptyInstance", func(t *testing.T) {
		t.Parallel()
		conf := &odem.Configuration{}
		args := getServiceRestartArgs(conf)
		if len(args) == 0 {
			t.Fatal("expected non-empty args")
		}
		// service name should end with @.service when instance is empty
		last := args[len(args)-1]
		if last != "odem@.service" {
			t.Errorf("service arg=%q, want %q", last, "odem@.service")
		}
	})
}

// --------------------------------------------------------------------------
// RestartService validation
// --------------------------------------------------------------------------

func TestRestartServiceRejectsEmptyInstance(t *testing.T) {
	t.Parallel()
	conf := &odem.Configuration{}
	// instance and remote both empty
	err := RestartService(context.Background(), conf)
	if err == nil {
		t.Fatal("expected error for empty deploy.instance, got nil")
	}
	if !strings.Contains(err.Error(), "instance") {
		t.Errorf("error should mention 'instance'; got: %v", err)
	}
}

func TestRestartServiceRejectsEmptyRemote(t *testing.T) {
	t.Parallel()
	conf := &odem.Configuration{}
	conf.Build.Deploy.Intstance = "prod"
	// remote empty
	err := RestartService(context.Background(), conf)
	if err == nil {
		t.Fatal("expected error for empty deploy.remote, got nil")
	}
	if !strings.Contains(err.Error(), "remote") {
		t.Errorf("error should mention 'remote'; got: %v", err)
	}
}
