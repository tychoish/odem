package release

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/exc"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/wpa"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/home"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/logger"
)

const ldFlagTmpl = `-ldflags=-s -w -X github.com/tychoish/odem/pkg/release.version=%s -X github.com/tychoish/odem.buildTime=%s`

func makeBaseCommand(ctx context.Context) *exc.Command {
	stderr := send.MakeWriterSender(logger.Plain(ctx).Sender())
	stderr.Store(level.Error)
	stdout := send.MakeWriterSender(logger.Plain(ctx).Sender())
	stderr.Store(level.Info)

	return new(exc.Command).
		WithStdOutput(stdout).
		WithStdError(stderr)
}

// BuildArtifacts builds release binaries for all configured targets,
// optionally compresses them with upx, generates sha256 checksums, and
// packages each binary into a zip (Windows) or tar.gz archive.
func BuildArtifacts(ctx context.Context) error {
	conf := odem.GetConfiguration(ctx)

	versionString := Version.Resolve().String()
	ldFlag := fmt.Sprintf(ldFlagTmpl, versionString, time.Now().Round(time.Millisecond).Format(time.RFC3339))

	var ec erc.Collector
	var jobs dt.List[fnx.Worker]

	namePart := fmt.Sprintf("%s-v%s", Name, versionString)
	versionBuildPath := filepath.Join(conf.Build.Path, namePart)

	basecmd := makeBaseCommand(ctx)
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

		binPath := filepath.Join(binaryPath, binaryName)

		steps := []*exc.Command{
			basecmd.Clone().
				WithName("go").
				WithArgs("build", ldFlag, "-o", binPath, "./cmd/odem.go").
				SetEnvVar("GOOS", build.GOOS).
				SetEnvVar("GOARCH", build.GOARCH),
			basecmd.Clone().Shell("bash", fmt.Sprintf("sha256sum %s > %s.sha256", binPath, filepath.Join(versionBuildPath, artifactName))),
		}

		if !conf.Build.DisableCompression {
			zpath := joindot(binaryPath, "lzma")
			if !conf.Runtime.DryRun {
				ec.Push(mkdirdashp(zpath))
			}
			zbin := filepath.Join(zpath, binaryName)

			upxcmd := basecmd.Clone().WithName("upx")
			if build.GOOS == "darwin" {
				upxcmd.WithArgs("--force-macos", "-q", "--lzma", binPath, "-o", zbin)
			} else {
				upxcmd.WithArgs("-q", "--lzma", binPath, "-o", zbin)
			}
			steps = append(steps,
				upxcmd,
				basecmd.Clone().Shell("bash", fmt.Sprintf("sha256sum %s > %s.lzma.sha256", binPath, filepath.Join(versionBuildPath, artifactName))),
			)

			if build.GOARCH == "386" {
				bin32 := filepath.Join(versionBuildPath, joinstr(Name, "32"))
				steps = append(steps,
					basecmd.Clone().WithName("cp").WithArgs(zbin, bin32),
					basecmd.Clone().Shell("bash", fmt.Sprintf("sha256sum %s > %s.sha256", bin32, bin32)),
				)
			}

			if build.GOOS == "darwin" && build.GOARCH == "arm64" {
				binapp := filepath.Join(versionBuildPath, joindot(Name, "app"))
				steps = append(steps,
					basecmd.Clone().WithName("cp").WithArgs(binPath, binapp),
					basecmd.Clone().Shell("bash", fmt.Sprintf("sha256sum %s > %s.sha256", binapp, binapp)),
				)
			}
		}

		switch build.GOOS {
		case "windows":
			zipball := filepath.Join(versionBuildPath, joindot(artifactName, "zip"))
			steps = append(steps,
				basecmd.Clone().WithName("zip").WithArgs("-j", zipball, binPath),
				basecmd.Clone().Shell("bash", fmt.Sprintf("sha256sum %s > %s.sha256", zipball, zipball)),
			)
		default:
			tarball := filepath.Join(versionBuildPath, joindot(artifactName, "tar.gz"))
			steps = append(steps,
				basecmd.Clone().WithName("tar").WithArgs("czvf", tarball, "-C", binaryPath, binaryName),
				basecmd.Clone().Shell("bash", fmt.Sprintf("sha256sum %s > %s.sha256", tarball, tarball)),
			)
		}

		if conf.Runtime.DryRun {
			for _, step := range steps {
				grip.Info(grip.KV("cmd", step.Name).KV("args", step.Args).KV("cmd", step))
			}
			continue
		} else {
			jobs.PushBack(func(ctx context.Context) error {
				for _, step := range steps {
					if err := step.Run(ctx); err != nil {
						return err
					}
				}
				return nil
			})
		}

	}
	if conf.Runtime.DryRun {
		return ec.Resolve()
	}

	ec.Push(wpa.RunWithPool(jobs.IteratorFront(), wpa.WorkerGroupConfDefaults()).Run(ctx))
	return ec.Resolve()
}

func LocalBuild(ctx context.Context) error {
	conf := odem.GetConfiguration(ctx)
	pwd := erc.Must(os.Getwd())
	for basepath := range irt.Keep(irt.Args(conf.Build.LocalRepoPath, home.TryExpandDirectory("~/src/odemp/"), pwd), infra.FileExists) {
		path := filepath.Join(basepath, "cmd", "odem.go")
		if !infra.FileExists(path) {
			continue
		}
		grip.Info(grip.MPrintln("building:", "./cmd/odem.go"))

		w := send.MakeWriterSender(logger.Plain(ctx).Sender())
		w.Store(level.Info)
		return makeBaseCommand(ctx).WithName("go").WithArgs("build", "./cmd/odem.go").WithStdOutput(w).WithStdError(w).Run(ctx)
	}
	return erc.Errorf("no odem checkout discoverable: %s", pwd)
}
