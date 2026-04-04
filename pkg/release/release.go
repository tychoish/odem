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
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/jasper"
	"github.com/tychoish/jasper/util"
	"github.com/tychoish/odem"
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

func GitDescribe() string {
	b := new(bytes.Buffer)
	buf := util.NewLocalBuffer(b)

	err := jasper.NewCommand().AppendArgs("git", "describe").SetOutputWriter(buf).Run(context.TODO())
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
	if !util.FileExists(buildDir) {
		return fmt.Errorf("build directory %q does not exist", buildDir)
	}

	artifacts := &dt.Set[string]{}
	if err := filepath.WalkDir(buildDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		name := d.Name()
		if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".sha256") {
			artifacts.Add(path)
		}

		return nil
	}); err != nil {
		return err
	}
	if zw := filepath.Join(buildDir, "windows-amd64.lzma", "odem.exe"); util.FileExists(zw) {
		artifacts.Add(joinstr(zw, "#odem for windows-amd64 with upx+lzma"))
	}
	if zw := filepath.Join(buildDir, "linux-amd64.lzma", "odem.exe"); util.FileExists(zw) {
		artifacts.Add(joinstr(zw, "#odem for linux-amd64 with upx+lzma"))
	}

	if artifacts.Len() == 0 {
		grip.Warningf("no artifacts found in %q", buildDir)
		return nil
	}

	args := irt.Collect(irt.Chain(irt.Args(irt.Args("gh", "release", "upload", releaseID, "--clobber"), artifacts.Iterator())))

	grip.Info(message.NewKV().
		KV("op", "upload artifacts").
		KV("release", releaseID).
		KV("tag", conf.Build.Tag).
		KV("num", artifacts.Len()).
		KV("args", args))

	return jasper.Context(ctx).CreateCommand(ctx).
		Add(args).
		SetCombinedSender(level.Info, logger.Plain(ctx).Sender()).
		Run(ctx)
}
