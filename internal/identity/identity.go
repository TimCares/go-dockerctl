package identity

import (
	"context"

	"go.uber.org/zap"
)

func CreateNewSOPSIdentity(ctx context.Context) error {
	zap.L().Info("creating new SOPS identity")
	return nil
}
