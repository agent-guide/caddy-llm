package configstore

import (
	"context"
	"fmt"

	"github.com/agent-guide/caddy-llm/llm/configstore/intf"
	"go.uber.org/zap"
)

var (
	configstoreCreators map[string]intf.ConfigStoreCreator
)

func init() {
	configstoreCreators = make(map[string]intf.ConfigStoreCreator)
}

func RegisterConfigStoreCreator(name string, creator intf.ConfigStoreCreator) {
	configstoreCreators[name] = creator
}

func CreateConfigStore(ctx context.Context, logger *zap.Logger, name string, config any) (intf.ConfigStorer, error) {
	creator, ok := configstoreCreators[name]
	if !ok {
		return nil, fmt.Errorf("unknown config store: %s", name)
	}
	return creator(ctx, logger, config)
}
