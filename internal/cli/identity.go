package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/identity"
)

var identityCommand = &cli.Command{
	Name:  "identity",
	Usage: "Manage SOPS identities",
	Commands: []*cli.Command{
		{
			Name:   "init",
			Usage:  "Create new Identity",
			Action: cliCreateNewSOPSIdentity,
		},
	},
}

func cliCreateNewSOPSIdentity(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load(cmd.String("config"), cmd.String("project"), cmd.String("env"))
	if err != nil {
		return err
	}

	identityPath, err := identity.GetSOPSIdentityPath(cfg)
	if err != nil {
		return err
	}

	_, err = identity.MaybeCreateNewSOPSIdentity(identityPath)
	return err
}
