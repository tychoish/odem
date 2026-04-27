package ep

import (
	"fmt"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/odemcli"
	"github.com/tychoish/odem/pkg/release"
)

func Build() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("build").
		Aliases("make").
		SetUsage("project automation and release tools").
		With(odemcli.CommandHelpAction).
		Flags(cmdr.FlagBuilder(false).
			SetName("dry-run", "n").
			SetUsage("disables all (most?) write operations for some (admin) operations").
			Flag()).
		With(odemcli.AttachConfiguration).
		Subcommanders(
			cmdr.MakeCommander().
				SetName(release.Name).
				SetUsage("build local binary").
				With(odemcli.ConfigurationAction(release.LocalBuild)),
			cmdr.MakeCommander().
				SetName("link").
				SetUsage("ensure a welformed symlink to a location in the sytsem path (as configured)").
				With(odemcli.ConfigurationAction(release.EnsureLink)),
			cmdr.MakeCommander().
				SetName("all").
				SetUsage("build artifacts for odem release; must run inside of the odem git repository").
				With(odemcli.ConfigurationAction(release.BuildArtifacts)),
			cmdr.MakeCommander().
				SetName("update").
				SetUsage("update the local repo/checkout").
				With(odemcli.ConfigurationAction(release.LocalUpdate)),
			cmdr.MakeCommander().
				SetName("release").
				SetUsage("release automation").
				With(odemcli.CommandHelpAction).
				Subcommanders(
					cmdr.MakeCommander().
						SetName("upload").
						SetUsage("upload built artifacts for the given release tag to GitHub").
						Flags(
							cmdr.FlagBuilder("").
								SetName("tag").
								AddAliases("t").
								SetUsage("git tag / version string to upload (e.g. v1.2.3)").
								SetRequired(true).
								Flag(),
						).
						With(odemcli.ConfigurationAction(release.UploadArtifacts)),
				),
			cmdr.MakeCommander().
				SetName("deploy").
				Aliases("push").
				SetUsage(fmt.Sprintf("deploy, and manage %s services (local or remote); require bootstrapping", release.Name)).
				With(odemcli.CommandHelpAction).
				Subcommanders(
					cmdr.MakeCommander().
						SetName("build").
						SetUsage("builds the release artifact; requires bootstrapping").
						With(odemcli.ConfigurationAction(release.BuildForDeploy)),
					cmdr.MakeCommander().
						SetName("update").
						SetUsage("updates the checkout; requires bootstrapping").
						With(odemcli.ConfigurationAction(release.BuildForDeploy)),
					cmdr.MakeCommander().
						SetName("service").
						SetUsage("update, build, and restart the service").
						With(odemcli.ConfigurationAction(release.Deploy)),
					cmdr.MakeCommander().
						SetName("restart").
						SetUsage("restart the service").
						With(odemcli.ConfigurationAction(release.RestartService)),
					cmdr.MakeCommander().
						SetName("db").
						Aliases("rebuild", "database").
						SetUsage("rebuild the database if new views have been built").
						With(odemcli.ConfigurationAction(db.RebuildCommand)),
				),
		)
}
