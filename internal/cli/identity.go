package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/identity"
)

func newIdentityCommand() *cli.Command {
	return &cli.Command{
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
}

// cliCreateNewSOPSIdentity prints the recipient to stdout, so it can be piped or captured.
func cliCreateNewSOPSIdentity(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.FromContext(ctx)
	if err != nil {
		return err
	}

	identityPath, err := identity.GetSOPSIdentityPath(cfg)
	if err != nil {
		return err
	}

	recipient, err := identity.MaybeCreateNewSOPSIdentity(ctx, identityPath)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.Root().Writer, recipient)
	return err
}
