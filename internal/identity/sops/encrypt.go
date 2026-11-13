package sops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TimCares/go-dockerctl/internal/utils"
	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/keyservice"
	"github.com/getsops/sops/v3/version"
)

// EncryptData encrypts plaintext bytes using the creation rule in .sops.yaml that matches path.
// The format (yaml, json, ...) is derived from the extension of path.
func EncryptData(creationRuleConfig *config.Config, store common.Store, filePath string, plaintext []byte) ([]byte, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	branches, err := store.LoadPlainFile(plaintext)
	if err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		return nil, errors.New("file must contain at least one document")
	}
	if store.HasSopsTopLevelKey(branches[0]) {
		return nil, errors.New("file is already encrypted")
	}

	tree := sops.Tree{
		Branches: branches,
		FilePath: absFilePath,
		Metadata: sops.Metadata{
			KeyGroups:               creationRuleConfig.KeyGroups,
			ShamirThreshold:         creationRuleConfig.ShamirThreshold,
			UnencryptedSuffix:       creationRuleConfig.UnencryptedSuffix,
			EncryptedSuffix:         creationRuleConfig.EncryptedSuffix,
			UnencryptedRegex:        creationRuleConfig.UnencryptedRegex,
			EncryptedRegex:          creationRuleConfig.EncryptedRegex,
			UnencryptedCommentRegex: creationRuleConfig.UnencryptedCommentRegex,
			EncryptedCommentRegex:   creationRuleConfig.EncryptedCommentRegex,
			MACOnlyEncrypted:        creationRuleConfig.MACOnlyEncrypted,
			Version:                 version.Version,
		},
	}

	dataKey, errs := tree.GenerateDataKeyWithKeyServices(
		[]keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	)
	if len(errs) > 0 {
		return nil, fmt.Errorf("generating data key: %v", errs)
	}

	err = common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &tree,
		Cipher:  aes.NewCipher(),
	})
	if err != nil {
		return nil, err
	}
	return store.EmitEncryptedFile(tree)
}

func Encrypt(sopsConfigFilePath string, filePath string) error {
	creationRulesConfig, sopsFileStore, err := SOPSFileInfo(sopsConfigFilePath, filePath)
	if err != nil {
		return fmt.Errorf("getting SOPS file info: %w", err)
	}

	contents, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading sops file %s: %w", filePath, err)
	}

	encryptedContents, err := EncryptData(creationRulesConfig, sopsFileStore, filePath, contents)
	if err != nil {
		return fmt.Errorf("encrypting file %s: %w", filePath, err)
	}

	return utils.AtomicWrite(filePath, encryptedContents, utils.SecretFilePerm)
}
