package release

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"time"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/exc"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/wpa"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/home"
	"github.com/tychoish/odem/pkg/logger"
)

const ldFlagTmpl = `-ldflags=-s -w -X github.com/tychoish/odem/pkg/release.version=%s -X github.com/tychoish/odem/pkg/release.buildTime=%s`

func makeBaseCommand(ctx context.Context) *exc.Command {
	o := send.MakeWriterSender(logger.Plain(ctx).Sender())
	o.Store(level.Info)
	e := send.MakeWriterSender(logger.Plain(ctx).Sender())
	e.Store(level.Error)

	return new(exc.Command).WithStdError(e).WithStdOutput(o)
}

// BuildArtifacts builds release binaries for all configured targets,
// optionally compresses them with upx, generates sha256 checksums, and
// packages each binary into a zip (Windows) or tar.gz archive.
func BuildArtifacts(ctx context.Context) error {
	conf := odem.GetConfiguration(ctx)

	for basepath := range basePathCandidates(conf) {
		basecmd := makeBaseCommand(ctx).WithDirectory(basepath)
		versionString := Version.Resolve().String()
		ldFlag := fmt.Sprintf(ldFlagTmpl, versionString, time.Now().Round(time.Millisecond).Format(time.RFC3339))

		var ec erc.Collector
		var jobs dt.List[fnx.Worker]

		namePart := fmt.Sprintf("%s-v%s", Name, versionString)
		versionBuildPath := filepath.Join(basepath, conf.Build.Path, namePart)

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
	return ers.New("no odem checkout discoverable")
}

func basePathCandidates(conf *odem.Configuration) iter.Seq[string] {
	pwd := erc.Must(os.Getwd())

	return irt.Keep(irt.Args(pwd, conf.Build.LocalRepoPath, home.TryExpandDirectory("~/src/odem/")), fileExists)
}

func LocalUpdate(ctx context.Context) error {
	conf := odem.GetConfiguration(ctx)

	for basepath := range basePathCandidates(conf) {
		return makeBaseCommand(ctx).WithDirectory(basepath).WithArgs("git", "pull", "origin", "main").Run(ctx)
	}
	return errors.New("could not find local environment for the release build to update")
}

func LocalBuild(ctx context.Context) error {
	conf := odem.GetConfiguration(ctx)
	curVersion := GitDescribe()

	for basepath := range basePathCandidates(conf) {
		path := filepath.Join(basepath, "cmd", "odem.go")
		if !fileExists(path) {
			continue
		}

		args := irt.Collect(irt.Args(
			// build + version setting
			"build", fmt.Sprintf(ldFlagTmpl, curVersion, time.Now().Round(time.Millisecond).Format(time.RFC3339)),
			// target:
			"-o", filepath.Join(basepath, conf.Build.Path, Name),
			// source
			filepath.Join(basepath, "cmd", "odem.go")))

		grip.Info(grip.MPrintln("building:", append([]string{"go"}, args...)))

		return makeBaseCommand(ctx).
			WithName("go").
			WithArgs(args...).
			Run(ctx)
	}

	return ers.New("no odem checkout discoverable")
}

func getBuildPath(conf *odem.Configuration) (string, error) {
	for basepath := range basePathCandidates(conf) {
		return filepath.Join(basepath, conf.Build.Path, Name), nil
	}
	return "", ers.ErrNotFound
}

func EnsureLink(ctx context.Context, conf *odem.Configuration) error {
	path, err := getBuildPath(conf)
	if err != nil {
		return err
	}

	if _, err := os.Stat(conf.Build.BinaryLink); !os.IsNotExist(err) {
		target, err := filepath.EvalSymlinks(conf.Build.BinaryLink)
		if err != nil {
			return ers.Wrap(err, "couldn't resolve link target for extant link")
		}
		if target != path {
			return fmt.Errorf("extant target %q does not point to expected path %s", target, conf.Build.BinaryLink)
		}
		return nil
	}
	return makeBaseCommand(ctx).WithArgs("sudo", "ln", "-s", conf.Build.BinaryLink, path).Run(ctx)

}
