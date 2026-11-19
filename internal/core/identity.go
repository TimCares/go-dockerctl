package core

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/TimCares/go-see"

	"github.com/TimCares/go-dockerctl/internal/appdirs"
	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/identity"
	"github.com/TimCares/go-dockerctl/internal/observability"
	"github.com/TimCares/go-dockerctl/internal/observability/events"
	"github.com/TimCares/go-dockerctl/internal/project"
	"github.com/TimCares/go-dockerctl/internal/utils"
)

type MaybeCreateNewAgeHostIdentitiesArgs struct {
	ProjectMetadata *project.ProjectMetadata
	Envs            []string
}

func MaybeCreateNewAgeHostIdentities(ctx context.Context, args MaybeCreateNewAgeHostIdentitiesArgs) ([]string, error) {
	recipients := make([]string, 0, len(args.Envs))

	err := observability.ForEachEnv(ctx, args.Envs, func(ctx context.Context, env string) error {
		identityPath, err := getAgeIdentityPath(args.ProjectMetadata, env)
		if err != nil {
			return err
		}

		// TODO: should we somehow support multi-file atomic writes? Currently, we only have per-file atomic writes.
		recipient, err := identity.MaybeCreateNewAgeIdentity(ctx, identityPath)
		if err != nil {
			return err
		}
		recipients = append(recipients, recipient)
		// Note: We do NOT emit some "IdentityCreated" event here.
		// Creating an identity is scoped to a project and env,
		// but it doesn't change project state until the key is added as a recipient,
		// so no event is emitted here.
		return nil
	})
	if err != nil {
		return nil, err
	}

	return recipients, nil
}

// getAgeIdentityPath returns the path of the age key file SOPS should use.
func getAgeIdentityPath(projectMetadata *project.ProjectMetadata, env string) (string, error) {
	envDir, err := appdirs.ProjectEnvDir(projectMetadata.Id, env)
	if err != nil {
		return "", fmt.Errorf("determining project env dir: %w", err)
	}
	return identity.AgeKeyFile(envDir), nil
}

type AddSOPSRecipientArgs struct {
	ProjectPath      string
	NamespaceConfigs []config.NamespaceConfig
	Envs             []string
	Recipient        string
}

func AddSOPSRecipient(ctx context.Context, args AddSOPSRecipientArgs) error {
	sopsFilePaths := []string{}

	rootParameterGroup := project.ParameterGroup{
		IsSecret: true,
		Envs:     args.Envs,
	}

	rootSOPSFilePaths := rootParameterGroup.IterFilesOrdered(args.Envs)
	for i := range rootSOPSFilePaths {
		rootSOPSFilePaths[i] = filepath.Join(project.SecretsDir(args.ProjectPath), rootSOPSFilePaths[i])
	}
	sopsFilePaths = append(sopsFilePaths, rootSOPSFilePaths...)

	for _, namespaceConfig := range args.NamespaceConfigs {
		namespace := project.NewNamespace(namespaceConfig.Name, namespaceConfig.Envs)

		// No full namepsace.Validate needed here.
		if err := namespace.ValidateLayout(namespaceConfig.Path); err != nil {
			return fmt.Errorf("validating filesystem layout of namespace %s: %w", namespace.Name, err)
		}

		namespaceSopsFilePaths, err := namespace.IterSecretFilesOrdered(args.Envs)
		if err != nil {
			return fmt.Errorf("iterating secret files of namespace %s: %w", namespace.Name, err)
		}

		for i := range namespaceSopsFilePaths {
			namespaceSopsFilePaths[i] = filepath.Join(project.SecretsDir(namespaceConfig.Path), namespaceSopsFilePaths[i])
		}

		sopsFilePaths = append(sopsFilePaths, namespaceSopsFilePaths...)
	}

	sopsFilePaths, err := utils.FilterFilesExist(sopsFilePaths)
	if err != nil {
		return err
	}

	sopsConfigPath := project.SOPSConfigFile(args.ProjectPath)

	creationRules := make([]string, 0, len(args.Envs))
	for _, env := range args.Envs {
		envCreationRule := identity.SOPSConfigCreationRule(env)
		creationRules = append(creationRules, envCreationRule)
	}

	funcImplArgs := identity.AddSOPSIdentityArgs{
		SopsConfigPath: sopsConfigPath,
		CreationRules:  creationRules,
		SopsFilePaths:  sopsFilePaths,
		Recipient:      args.Recipient,
	}

	if err := identity.AddSOPSIdentity(ctx, funcImplArgs); err != nil {
		return err
	}

	// One SOPS update covers all envs, but the recipient is added once per env.
	for _, env := range args.Envs {
		see.Emit(observability.WithEnv(ctx, env), events.IdentityAdded{
			KeySource: identity.Age,
			Recipient: args.Recipient,
		})
	}
	return nil
}
