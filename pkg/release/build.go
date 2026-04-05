package release

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/wpa"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/jasper"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/logger"
)

const ldFlagTmpl = `-ldflags=-s -w -X github.com/tychoish/odem/pkg/release.version=%s -X github.com/tychoish/odem.buildTime=%s`

// BuildArtifacts builds release binaries for all configured targets,
// optionally compresses them with upx, generates sha256 checksums, and
// packages each binary into a zip (Windows) or tar.gz archive.
func BuildArtifacts(ctx context.Context) error {
	conf := odem.GetConfiguration(ctx)

	versionString := Version.Resolve().String()
	ldFlag := fmt.Sprintf(ldFlagTmpl, versionString, time.Now().Round(time.Millisecond).Format(time.RFC3339))

	var ec erc.Collector
	jpm := jasper.Context(ctx)
	var jobs dt.List[fnx.Worker]

	namePart := fmt.Sprintf("%s-v%s", Name, versionString)
	versionBuildPath := filepath.Join(conf.Build.Path, namePart)

	for build := range irt.Slice(conf.Build.Targets) {
		buildName := joindash(build.GOOS, build.GOARCH)
		binaryPath := filepath.Join(versionBuildPath, buildName)
		if !conf.Runtime.DryRun {
			ec.Push(mkdirdashp(binaryPath))
		}

		artifactName := fmt.Sprintf("%s-%s-%s-%s", Name, versionString, build.GOOS, build.GOARCH)
		var binaryName string
		if build.GOOS == "windows" {
			binaryName = joindot(Name, "exe")
		} else {
			binaryName = Name
		}
		grip.Debugln(ldFlag)
		cmd := jpm.CreateCommand(ctx).
			ID(binaryPath).
			AppendArgs("go", "build", ldFlag, "-o", filepath.Join(binaryPath, binaryName), "./cmd/odem.go").
			AddEnv("GOOS", build.GOOS).
			AddEnv("GOARCH", build.GOARCH).
			RedirectOutputToError(true).
			SetOutputSender(level.Debug, logger.Plain(ctx).Sender()).
			SetErrorSender(level.Error, logger.Plain(ctx).Sender())

		cmd.Sh(fmt.Sprintf("sha256sum %s > %s.sha256", filepath.Join(binaryPath, binaryName), filepath.Join(versionBuildPath, artifactName)))

		if !conf.Build.DisableCompression {
			zpath := joindot(binaryPath, "lzma")
			if !conf.Runtime.DryRun {
				ec.Push(mkdirdashp(zpath))
			}
			zbin := filepath.Join(zpath, binaryName)
			if build.GOOS == "darwin" {
				cmd.AppendArgs("upx", "--force-macos", "-q", "--lzma", filepath.Join(binaryPath, binaryName), "-o", zbin)
			} else {
				cmd.AppendArgs("upx", "-q", "--lzma", filepath.Join(binaryPath, binaryName), "-o", zbin)
			}
			cmd.Sh(fmt.Sprintf("sha256sum %s > %s.lzma.sha256", filepath.Join(binaryPath, binaryName), filepath.Join(versionBuildPath, artifactName)))
			if build.GOARCH == "386" {
				bin32 := filepath.Join(versionBuildPath, joinstr(Name, "32"))
				cmd.AppendArgs("cp", zbin, bin32)
				cmd.Sh(fmt.Sprintf("sha256sum %s > %s.sha256", bin32, bin32))
			}

			if build.GOOS == "darwin" && build.GOARCH == "arm64" {
				binapp := filepath.Join(versionBuildPath, joindot(Name, "app"))
				cmd.AppendArgs("cp", filepath.Join(binaryPath, binaryName), binapp)
				cmd.Sh(fmt.Sprintf("sha256sum %s > %s.sha256", binapp, binapp))

			}

		}
		switch {
		case build.GOOS == "windows":
			zipball := filepath.Join(versionBuildPath, joindot(artifactName, "zip"))
			cmd.AppendArgs("zip", "-j", zipball, filepath.Join(binaryPath, binaryName))
			cmd.Sh(fmt.Sprintf("sha256sum %s > %s.sha256", zipball, zipball))
		default:
			tarball := filepath.Join(versionBuildPath, joindot(artifactName, "tar.gz"))
			cmd.AppendArgs("tar", "czvf", tarball, "-C", binaryPath, binaryName)
			cmd.Sh(fmt.Sprintf("sha256sum %s > %s.sha256", tarball, tarball))
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
