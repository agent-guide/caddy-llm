package intf

import (
	"context"
)

type ConfigObjectDecoder func(data []byte) (any, error)

type ConfigStorer interface {
	GetCredentialStore(ctx context.Context, decodeConfigObject ConfigObjectDecoder) (CredentialStorer, error)

	GetProviderConfigStore(ctx context.Context, decodeProviderConfig ConfigObjectDecoder) (ProviderConfigStorer, error)

	GetLocalAPIKeyStore(ctx context.Context, decodeLocalAPIKey ConfigObjectDecoder) (LocalAPIKeyStorer, error)

	GetRouteStore(ctx context.Context, decodeRoute ConfigObjectDecoder) (RouteStorer, error)
}
