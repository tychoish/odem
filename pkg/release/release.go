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

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/grip"
	"github.com/tychoish/jasper"
	"github.com/tychoish/jasper/util"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/infra"
)

var (
	version   string
	buildTime string
)

func GetVersion() string      { return version }
func GetBuildTime() time.Time { return erc.Must(time.Parse(time.RFC3339, buildTime)) }
func GitDescribe(ctx context.Context) string {
	b := new(bytes.Buffer)
	buf := util.NewLocalBuffer(b)

	err := jasper.Context(ctx).CreateCommand(ctx).AppendArgs("git", "describe").SetOutputWriter(buf).Run(ctx)
	grip.Warning(ers.Wrap(err, "git describe for release versioning"))

	return cmp.Or(string(bytes.TrimSpace(b.Bytes())), "<UNKNOWN>")
}

// UploadArtifacts uploads all .zip, .tar.gz, and .sha256 files found in the
// build directory for the given tag to the matching GitHub release using
// `gh release upload`.
func UploadArtifacts(ctx context.Context, tag string) fnx.Worker {
	conf := odem.GetConfiguration(ctx)

	buildDir := filepath.Join(conf.Build.Path, tag)
	if !util.FileExists(buildDir) {
		return infra.ErrWorker(fmt.Errorf("build directory %q does not exist", buildDir))
	}

	var artifacts []string
	if err := filepath.WalkDir(buildDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".sha256") {
			artifacts = append(artifacts, path)
		}
		return nil
	}); err != nil {
		return infra.ErrWorker(err)
	}

	if len(artifacts) == 0 {
		grip.Infof("no artifacts found in %q", buildDir)
		return infra.ErrWorker(nil)
	}

	grip.Infof("uploading %d artifacts for %s", len(artifacts), tag)
	args := append([]string{"gh", "release", "upload", tag, "--clobber"}, artifacts...)
	return jasper.Context(ctx).CreateCommand(ctx).AppendArgs(args...).Worker()
}
