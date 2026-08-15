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
	cfg, configErr := config.GetConfig(cmd.String("config"), cmd.String("project"), cmd.String("env"))
	if configErr != nil {
		return configErr
	}

	identityPath := identity.GetSOPSIdentityPath(*cfg)
	_, identityErr := identity.CreateNewSOPSIdentity(identityPath)
	if identityErr != nil {
		return identityErr
	}

	return nil
}
