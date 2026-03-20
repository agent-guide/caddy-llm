package intf

import (
	"context"

	"go.uber.org/zap"
)

type ConfigStorer interface {
	GetCredentialStore() CredentialStorer

	GetProviderConfigStore() ProviderConfigStorer

	GetVXApiKeyStore() VXApiKeyStorer
}

type ConfigStoreCreator func(ctx context.Context, logger *zap.Logger, config any) (ConfigStorer, error)
