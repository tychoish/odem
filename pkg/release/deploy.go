package release

import (
	"context"
	"fmt"

	"github.com/tychoish/grip"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/infra"
)

func Deploy(ctx context.Context, conf *odem.Configuration) error {
	if err := conf.ValidateDeploy(); err != nil {
		return err
	}
	grip.Notice(grip.When(conf.Runtime.Hostname == "", "cannot determine hostname, will proceed as if the deploy is remote"))

	if err := BuildForDeploy(ctx, conf); err != nil {
		return err
	}

	return RestartService(ctx, conf)
}

func UpdateForDeploy(ctx context.Context, conf *odem.Configuration) error {
	if conf.Build.Deploy.Remote == conf.Runtime.Hostname {
		grip.Info(grip.KV("op", "updating").KV("host", "local"))
		return LocalUpdate(ctx, conf)
	}

	grip.Info(grip.KV("op", "rebuilding").KV("host", conf.Build.Deploy.Remote))
	return infra.Command(ctx).SSH(conf.Build.Deploy.Remote, Name, "update").Run(ctx)
}

func RestartService(ctx context.Context, conf *odem.Configuration) error {
	if err := conf.ValidateDeploy(); err != nil {
		return err
	}
	srvRestartArgs := getServiceRestartArgs(conf)

	if conf.Build.Deploy.Remote == conf.Runtime.Hostname {
		grip.Info(grip.KV("op", "restarting service").KV("host", "local").KV("args", srvRestartArgs))
		return infra.Command(ctx).WithArgs(srvRestartArgs...).Run(ctx)
	}

	grip.Info(grip.KV("op", "restarting service").KV("host", conf.Build.Deploy.Remote).KV("args", srvRestartArgs))
	return infra.Command(ctx).SSH(conf.Build.Deploy.Remote, srvRestartArgs...).Run(ctx)
}

func BuildForDeploy(ctx context.Context, conf *odem.Configuration) error {
	if conf.Build.Deploy.Remote == conf.Runtime.Hostname {
		grip.Info(grip.KV("op", "rebuilding").KV("host", "local"))
		return LocalBuild(ctx, conf)
	}

	grip.Info(grip.KV("op", "rebuilding").KV("host", conf.Build.Deploy.Remote))
	return infra.Command(ctx).SSH(conf.Build.Deploy.Remote, Name, "build").Run(ctx)
}

func getServiceRestartArgs(conf *odem.Configuration) []string {
	if conf.Build.Deploy.GlobalService {
		return []string{"sudo", "systemctl", "restart", fmt.Sprintf("%s@%s.service", Name, conf.Build.Deploy.Intstance)}
	}
	return []string{"systemctl", "--user", "restart", fmt.Sprintf("%s@%s.service", Name, conf.Build.Deploy.Intstance)}
}
