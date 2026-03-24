package intf

import (
	"context"
)

type ConfigObjectDecoder func(data []byte) (any, error)

type ConfigStorer interface {
	GetCredentialStore(ctx context.Context, decodeConfigObject ConfigObjectDecoder) (CredentialStorer, error)

	GetProviderConfigStore() ProviderConfigStorer

	GetVXApiKeyStore() VXApiKeyStorer
}
