package release

import (
	"github.com/masterminds/semver"
	"github.com/tychoish/fun/ers"
)

func ValidateVersion(tag string) error {
	return ers.Wrapf(ignorevalue(semver.NewVersion(tag)), "could not parse version from %q", tag)
}

func ignorevalue[T any](_ T, err error) error { return err }
