package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/TimCares/go-infractl/internal/identity"
)

var identityCommand = &cli.Command{
	Name:  "identity",
	Usage: "Manage SOPS identities",
	Commands: []*cli.Command{
		{
			Name:   "create",
			Usage:  "Create new Identity",
			Action: cliCreateNewSOPSIdentity,
		},
	},
}

func cliCreateNewSOPSIdentity(ctx context.Context, cmd *cli.Command) error {
	return identity.CreateNewSOPSIdentity(ctx)
}
