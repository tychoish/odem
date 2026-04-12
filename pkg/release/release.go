package release

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/masterminds/semver"
	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/exc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/logger"
)

const Name = "odem"

var (
	version   string
	buildTime string
	Version   adt.Once[*semver.Version]
	BuildTime adt.Once[time.Time]
)

func init() {
	Version.Set(func() *semver.Version { return erc.Must(semver.NewVersion(cmp.Or(version, GitDescribe()))) })
	BuildTime.Set(func() time.Time { return erc.Must(time.Parse(time.DateTime, cmp.Or(buildTime, "1986-05-19 00:00:00"))) })
}

func IsPrerelease(version string) bool {
	return erc.Must(semver.NewVersion(version)).Prerelease() != ""
}

func GitDescribe() string {
	b := new(bytes.Buffer)

	err := new(exc.Command).WithName("git").WithArgs("describe").WithStdOutput(b).Run(context.TODO())
	grip.Warning(ers.Wrap(err, "git describe for release versioning"))

	return cmp.Or(string(bytes.TrimSpace(b.Bytes())), "<UNKNOWN>")
}

// UploadArtifacts uploads all .zip, .tar.gz, and .sha256 files found in the
// build directory for the given tag to the matching GitHub release using
// `gh release upload`.
func UploadArtifacts(ctx context.Context, conf *odem.Configuration) error {
	var releaseID string
	if strings.HasPrefix(conf.Build.Tag, joindash(Name, "")) {
		releaseID = conf.Build.Tag[len(Name)+1:]
	} else {
		releaseID = conf.Build.Tag
	}

	buildDir := filepath.Join(conf.Build.Path, conf.Build.Tag)
	if !infra.FileExists(buildDir) {
		return fmt.Errorf("build directory %q does not exist", buildDir)
	}

	artifacts := &dt.Set[string]{}
	if err := filepath.WalkDir(buildDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		name := d.Name()
		switch {
		case strings.Contains(name, "386"):
			break
		case strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz"):
			artifacts.Add(joinstr(path, "#archive: "))
		case strings.HasSuffix(name, ".sha256"):
			artifacts.Add(joinstr(path, "#checksum (sha256): ", name))
		}

		return nil
	}); err != nil {
		return err
	}
	if zw := filepath.Join(buildDir, "windows-amd64.lzma", joindot(Name, "exe")); infra.FileExists(zw) {
		artifacts.Add(joinstr(zw, "#binary (+upx+lzma): ", zw))
	}
	if zw := filepath.Join(buildDir, "linux-amd64.lzma", Name); infra.FileExists(zw) {
		artifacts.Add(joinstr(zw, "#binary (+upx+lzma): ", zw))
	}
	if zw := filepath.Join(buildDir, joinstr(Name, "32")); infra.FileExists(zw) {
		artifacts.Add(joinstr(zw, "#binary (+upx+lzma) linux-32bit: ", zw))
	}
	if zw := filepath.Join(buildDir, joindot(Name, ".app")); infra.FileExists(zw) {
		artifacts.Add(joinstr(zw, "#binary (+upx+lzma) macOS : ", zw))
	}

	if artifacts.Len() == 0 {
		grip.Warning(grip.MPrintf("no artifacts found in %q", buildDir))
		return nil
	}

	args := irt.Collect(irt.Chain(irt.Args(irt.Args("gh", "release", "upload", releaseID, "--clobber"), artifacts.Iterator())))

	grip.Info(grip.KV("op", "upload artifacts").
		KV("release", releaseID).
		KV("tag", conf.Build.Tag).
		KV("num", artifacts.Len()).
		KV("args", args))

	w := send.MakeWriterSender(logger.Plain(ctx).Sender())
	w.Store(level.Info)
	return new(exc.Command).WithName(args[0]).WithArgs(args[1:]...).WithStdOutput(w).WithStdError(w).Run(ctx)
}
