package release

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/wpa"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/jasper"
	"github.com/tychoish/jasper/util"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/logger"
)

const ldFlagTmpl = `-ldflags=-s -w -X github.com/tychoish/odem/pkg/release.version=%s -X github.com/tychoish/odem.buildTime=%s`

// BuildArtifacts builds release binaries for all configured targets,
// optionally compresses them with upx, generates sha256 checksums, and
// packages each binary into a zip (Windows) or tar.gz archive.
func BuildArtifacts(ctx context.Context) error {
	conf := odem.GetConfiguration(ctx)

	versionString := GitDescribe(ctx)
	grip.Infoln("🤖 🎶", "odem", versionString)
	ldFlag := fmt.Sprintf(ldFlagTmpl, versionString, time.Now().Round(time.Millisecond).Format(time.RFC3339))

	var ec erc.Collector
	jpm := jasper.Context(ctx)
	var jobs dt.List[fnx.Worker]

	for build := range irt.Slice(conf.Build.Targets) {
		binaryPath := filepath.Join(conf.Build.Path, versionString, joindot(build.GOOS, build.GOARCH))
		if !conf.Runtime.DryRun {
			ec.Push(mkdirdashp(binaryPath))
		}

		var binaryName string
		if build.GOOS == "windows" {
			binaryName = "odem.exe"
		} else {
			binaryName = "odem"
		}

		cmd := jpm.CreateCommand(ctx).
			ID(binaryPath).
			AppendArgs("go", "build", ldFlag, "-o", filepath.Join(binaryPath, binaryName), "./cmd/odem.go").
			AddEnv("GOOS", build.GOOS).
			AddEnv("GOARCH", build.GOARCH).
			RedirectOutputToError(true).
			SetOutputSender(level.Debug, logger.Plain(ctx).Sender()).
			SetErrorSender(level.Error, logger.Plain(ctx).Sender())

		if !conf.Build.DisableCompression && build.GOOS != "darwin" {
			zpath := joindot(binaryPath, "lzma")
			if !conf.Runtime.DryRun {
				ec.Push(mkdirdashp(zpath))
			}
			cmd.AppendArgs("upx", "-q", "--lzma",
				filepath.Join(binaryPath, binaryName),
				"-o", filepath.Join(zpath, binaryName))
		}

		cmd.Sh(fmt.Sprintf("cd %q && sha256sum %s > %s.sha256", binaryPath, binaryName, binaryName))

		archiveName := fmt.Sprintf("odem-%s-%s-%s", versionString, build.GOOS, build.GOARCH)
		if build.GOOS == "windows" {
			cmd.AppendArgs("zip", "-j",
				filepath.Join(binaryPath, joindot(archiveName, ".zip")),
				filepath.Join(binaryPath, binaryName))
		} else {
			cmd.AppendArgs("tar", "czvf",
				filepath.Join(binaryPath, joindot(archiveName, ".tar.gz")),
				"-C", binaryPath, binaryName)
		}

		if conf.Runtime.DryRun {
			grip.Info(cmd.String())
			continue
		}

		jobs.PushBack(cmd.Worker())
	}

	ec.Push(wpa.RunWithPool(jobs.IteratorFront(), wpa.WorkerGroupConfDefaults()).Run(ctx))
	return ec.Resolve()
}

func joindot(s ...string) string { return strings.Join(s, ".") }

func mkdirdashp(path string) error {
	if util.FileExists(path) {
		return nil
	}
	grip.Infof("making directory %q", path)
	return os.MkdirAll(path, 0o766)
}
