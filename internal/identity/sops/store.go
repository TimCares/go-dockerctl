package sops

import (
	"fmt"
	"path/filepath"

	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
)

func SOPSFileInfo(sopsConfigFilePath string, sopsFilePath string) (*config.Config, common.Store, error) {
	absSOPSFilePath, err := filepath.Abs(sopsFilePath)
	if err != nil {
		return nil, nil, err
	}

	cfg, err := config.LoadCreationRuleForFile(sopsConfigFilePath, absSOPSFilePath, nil)
	if err != nil {
		return nil, nil, err
	}
	if cfg == nil {
		return nil, nil, fmt.Errorf("%s has no creation_rules", sopsConfigFilePath)
	}
	storesConf, err := config.LoadStoresConfig(sopsConfigFilePath)
	if err != nil {
		return nil, nil, err
	}
	store := common.DefaultStoreForPath(storesConf, absSOPSFilePath)

	return cfg, store, nil
}
