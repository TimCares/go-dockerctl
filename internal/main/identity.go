package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/identity"
	"github.com/TimCares/go-dockerctl/internal/manager"
	"github.com/TimCares/go-dockerctl/internal/observability/events"
	"github.com/TimCares/go-dockerctl/internal/project"
	"github.com/TimCares/go-dockerctl/internal/utils"
	"github.com/TimCares/go-see"
)

type MaybeCreateNewAgeHostIdentitiesArgs struct {
	ProjectMetadata *project.ProjectMetadata
	Envs            []string
}

func MaybeCreateNewAgeHostIdentities(ctx context.Context, args MaybeCreateNewAgeHostIdentitiesArgs) ([]string, error) {

	recipients := make([]string, len(args.Envs))

	for _, env := range args.Envs {
		identityPath, err := getAgeIdentityPath(args.ProjectMetadata, env)
		if err != nil {
			return nil, err
		}

		// TODO: should we somehow support multi-file atomic writes? Currently, we only have per-file atomic writes.
		recipient, err := identity.MaybeCreateNewAgeIdentity(ctx, identityPath)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, recipient)
		see.Emit(ctx, events.IdentityCreated{
			Hostname:     utils.Hostname(),
			IdentityPath: identityPath,
			Recipient:    recipient,
		})
	}

	return recipients, nil

}

// getAgeIdentityPath returns the path of the age key file SOPS should use.
func getAgeIdentityPath(projectMetadata *project.ProjectMetadata, env string) (string, error) {
	projectConfigDir, err := manager.DockerctlProjectConfigDir(projectMetadata)
	if err != nil {
		return "", fmt.Errorf("determining project config dir: %w", err)
	}
	return filepath.Join(projectConfigDir, env, identity.DefaultAgeKeyFileSubpath), nil
}

type AddSOPSIdentityArgs struct {
	ProjectPath      string
	NamespaceConfigs []config.NamespaceConfig
	Envs             []string
	Recipient        string
}

func AddSOPSIdentity(ctx context.Context, args AddSOPSIdentityArgs) error {
	sopsFilePaths := []string{}

	rootParameterGroup := project.ParameterGroup{
		IsSecret: true,
		Envs:     args.Envs,
	}

	rootSOPSFilePaths := rootParameterGroup.IterFilesOrdered(args.Envs)
	for i := range rootSOPSFilePaths {
		rootSOPSFilePaths[i] = filepath.Join(args.ProjectPath, project.SecretsDirName, rootSOPSFilePaths[i])
	}
	sopsFilePaths = append(sopsFilePaths, rootSOPSFilePaths...)

	for _, namespaceConfig := range args.NamespaceConfigs {
		namespace := project.NewNamespace(namespaceConfig.Name, namespaceConfig.Envs)

		if err := namespace.ValidateLayout(namespaceConfig.Path); err != nil {
			return fmt.Errorf("validating filesystem layout of namespace %s: %w", namespace.Name, err)
		}

		namespaceSopsFilePaths, err := namespace.IterSecretFilesOrdered(args.Envs)
		if err != nil {
			return fmt.Errorf("iterating secret files of namespace %s: %w", namespace.Name, err)
		}

		for i := range namespaceSopsFilePaths {
			namespaceSopsFilePaths[i] = filepath.Join(namespaceConfig.Path, project.SecretsDirName, namespaceSopsFilePaths[i])
		}

		sopsFilePaths = append(sopsFilePaths, namespaceSopsFilePaths...)
	}

	sopsFilePaths, err := utils.FilterFilesExist(sopsFilePaths)
	if err != nil {
		return err
	}

	sopsConfigPath := filepath.Join(args.ProjectPath, identity.SOPSConfigFileName)

	creationRules := make([]string, len(args.Envs))
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

	hostname := utils.Hostname()
	for _, env := range args.Envs {
		// TODO: hostname (and env) will be included in almost every event + their metrics, so improve this redundancy.
		// Same thing is about the current project (UUID) that we operate on!
		see.Emit(ctx, events.IdentityAdded{
			Hostname:  hostname,
			Env:       env,
			KeySource: identity.Age,
			Recipient: args.Recipient,
		})
	}
	return nil
}
