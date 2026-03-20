package intf

import "context"

type VXApiKeyStorer interface {
	ListByProviderConfigID(ctx context.Context, providerId string) ([]any, error)

	Save(ctx context.Context, key string, providerId string, obj any) error

	Update(ctx context.Context, key string, obj any) error

	Delete(ctx context.Context, key string) error

	Get(ctx context.Context, key string) (any, error)
}
