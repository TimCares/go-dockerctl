package sops

import (
	"github.com/getsops/sops/v3/cmd/sops/subcommand/updatekeys"
	"github.com/getsops/sops/v3/keyservice"
)

func UpdateKeys(sopsConfigFilePath string, sopsFilePath string) error {
	opts := updatekeys.Opts{
		InputPath:       sopsFilePath,
		ShamirThreshold: 0, // 0 -> SOPS default value, means to keep the file's current one. TODO: maybe later surface in API.
		KeyServices:     []keyservice.KeyServiceClient{keyservice.NewLocalClient()},
		Interactive:     false, // Always automatic.
		ConfigPath:      sopsConfigFilePath,
		InputType:       "yaml", // Opinionated: We only support yaml.
	}
	return updatekeys.UpdateKeys(opts)
}
