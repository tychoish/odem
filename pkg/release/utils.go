package release

import (
	"os"
	"strings"

	"github.com/masterminds/semver"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/grip"
	"github.com/tychoish/jasper/util"
)

func ValidateVersion(tag string) error {
	if strings.HasPrefix(tag, joindash(Name, "")) {
		tag = tag[len(Name)+1:]
	}

	return ers.Wrapf(ignorevalue(semver.NewVersion(tag)), "could not parse version from %q", tag)
}

func ignorevalue[T any](_ T, err error) error { return err }
func joinstr(s ...string) string              { return strings.Join(s, "") }
func joindot(s ...string) string              { return strings.Join(s, ".") }
func joindash(s ...string) string             { return strings.Join(s, "-") }

func mkdirdashp(path string) error {
	if util.FileExists(path) {
		return nil
	}
	grip.Info(grip.MPrintf("making directory %q", path))
	return os.MkdirAll(path, 0o766)
}
