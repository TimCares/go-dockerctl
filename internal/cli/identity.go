package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/TimCares/go-dockerctl/internal/core"
)

func newIdentityCommand() *cli.Command {
	return &cli.Command{
		Name:  "identity",
		Usage: "Manage SOPS age identities",
		Commands: []*cli.Command{
			{
				Name:   "create",
				Usage:  "Create a new age host identity",
				Action: withProject(cliCreateNewAgeHostIdentity),
			},
			{
				Name:   "add",
				Usage:  "Add an age Identity as a recipient",
				Action: withProject(cliAddSOPSRecipient),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "recipient",
						Usage:       "Age public key",
						DefaultText: "age1...",
					},
				},
			},
		},
	}
}

// cliCreateNewAgeHostIdentity prints the recipient to stdout, so it can be piped or captured.
func cliCreateNewAgeHostIdentity(ctx context.Context, cmd *cli.Command, p Project) error {
	args := core.MaybeCreateNewAgeHostIdentitiesArgs{
		ProjectMetadata: p.Metadata,
		Envs:            p.Envs,
	}

	recipients, err := core.MaybeCreateNewAgeHostIdentities(ctx, args)
	if err != nil {
		return err
	}

	writer := cmd.Root().Writer
	for _, recipient := range recipients {
		if _, err := fmt.Fprintln(writer, recipient); err != nil {
			return err
		}
	}
	return nil
}

func cliAddSOPSRecipient(ctx context.Context, cmd *cli.Command, p Project) error {
	recipient := cmd.String("recipient")
	if recipient == "" {
		return fmt.Errorf("no recipient provided")
	}

	args := core.AddSOPSRecipientArgs{
		ProjectPath:      p.Settings.ProjectPath,
		NamespaceConfigs: p.Config.Namespaces,
		Envs:             p.Envs,
		Recipient:        recipient,
	}

	if err := core.AddSOPSRecipient(ctx, args); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.Root().Writer, recipient)
	return err
}
