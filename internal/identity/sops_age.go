package identity

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/TimCares/go-dockerctl/internal/identity/sops"
	"github.com/TimCares/go-dockerctl/internal/utils"
	"github.com/TimCares/go-see"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// SOPSKeySources are the kinds of key SOPS can encrypt secrets for.
type SOPSKeySources string

const (
	// Age is an age X25519 key.
	Age SOPSKeySources = "age"
	// PGP is a PGP / GnuPG key.
	PGP SOPSKeySources = "pgp"
	// KMS is any Key Management System, e.g. HashiCorp Vault.
	KMS SOPSKeySources = "kms"
)

func addAgeRecipientToSOPSConfig(ctx context.Context, configPath string, creationRules []string, recipient string) error {
	ctx = context.WithValue(ctx, "sopsConfigPath", configPath)

	rawConfig, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading sops config file: %w", err)
	}

	var sopsConfig sops.Config

	if err := yaml.Unmarshal(rawConfig, &sopsConfig); err != nil {
		return fmt.Errorf("decoding sops config file: %w", err)
	}

	creationRuleRegexesSet := make(map[string]struct{}, len(creationRules))

	for _, creationRuleRegex := range creationRules {
		creationRuleRegexesSet[creationRuleRegex] = struct{}{}
	}

	var writeRequired = false

	for i := range sopsConfig.CreationRules {
		creationRule := &sopsConfig.CreationRules[i]

		if _, ok := creationRuleRegexesSet[creationRule.PathRegex]; ok {
			if !slices.Contains(creationRule.Age, recipient) {
				creationRule.Age = append(creationRule.Age, recipient)
				see.L(ctx).Info("Added recipient to creation rule", zap.String("creationRule", creationRule.PathRegex))
				writeRequired = true
			} else {
				see.L(ctx).Info("Creation rule already has recipient", zap.String("creationRule", creationRule.PathRegex))
			}
		}
	}

	if !writeRequired {
		return nil
	}

	updatedConfig, err := yaml.Marshal(&sopsConfig)
	if err != nil {
		return fmt.Errorf("encoding sops config file: %w", err)
	}

	if err := utils.AtomicWrite(configPath, updatedConfig, utils.UserFilePerm); err != nil {
		return fmt.Errorf("writing sops config file: %w", err)
	}

	return nil
}

func SOPSConfigCreationRule(env string) string {
	return fmt.Sprintf("\\.%s\\.yaml$", env) // "\.prod\.yaml$" => Matches any "^*\.prod\.yaml$".
}
