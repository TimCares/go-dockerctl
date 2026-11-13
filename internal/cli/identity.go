package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/main"
	"github.com/TimCares/go-dockerctl/internal/project"
)

func newIdentityCommand() *cli.Command {
	return &cli.Command{
		Name:  "identity",
		Usage: "Manage SOPS age identities",
		Commands: []*cli.Command{
			{
				Name:   "create",
				Usage:  "Create a new age host identity",
				Action: cliCreateNewAgeHostIdentity,
			},
			{
				Name:   "add",
				Usage:  "Add an age Identity as a recipient",
				Action: cliAddSOPSIdentity,
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
func cliCreateNewAgeHostIdentity(ctx context.Context, cmd *cli.Command) error {
	// We do NOT need config loading and validation here!!!
	// TODO: change beforeRoot!

	projectPath := cmd.String("project-path")
	envs := cmd.StringSlice("envs") // beforeRoot validates correct naming already.

	projectMetaData, err := project.ReadProjectMetaFile(projectPath)
	if err != nil {
		return fmt.Errorf("reading project metadata: %w", err)
	}

	args := main.MaybeCreateNewAgeHostIdentitiesArgs{
		ProjectMetadata: projectMetaData,
		Envs:            envs,
	}

	recipients, err := main.MaybeCreateNewAgeHostIdentities(ctx, args)

	writer := cmd.Root().Writer
	for _, recipient := range recipients {
		_, err = fmt.Fprintln(writer, recipient)

		if err != nil {
			return err
		}
	}
	return nil
}

func cliAddSOPSIdentity(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.ConfigFromContext(ctx)
	if err != nil {
		return err
	}

	projectPath := cmd.String("project-path")
	envs := cmd.StringSlice("envs") // beforeRoot validates correct naming already.

	recipient := cmd.String("recipient")
	if recipient == "" {
		return fmt.Errorf("no recipient provided")
	}

	args := main.AddSOPSIdentityArgs{
		ProjectPath:      projectPath,
		NamespaceConfigs: cfg.Namespaces,
		Envs:             envs,
		Recipient:        recipient,
	}

	if err := main.AddSOPSIdentity(ctx, args); err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.Root().Writer, recipient)
	return err
}
