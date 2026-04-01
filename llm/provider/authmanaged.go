package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/agent-guide/caddy-agent-gateway/llm/cliauth/credential"
	"github.com/agent-guide/caddy-agent-gateway/llm/cliauth/manager"
	"github.com/cloudwego/eino/schema"
)

type authManagedProvider struct {
	base         Provider
	providerName string
	authManager  *manager.Manager
	config       ProviderConfig
}

// WrapWithAuthManager decorates a provider with auth-manager driven credential
// selection and execution result feedback. The handler layer then only needs to call the
// returned provider instance.
func WrapWithAuthManager(base Provider, providerName string, authMgr *manager.Manager) Provider {
	if base == nil || authMgr == nil {
		return base
	}
	cfg := base.Config()
	if cfg.Name == "" {
		cfg.Name = providerName
	}
	cfg.Defaults()
	return &authManagedProvider{
		base:         base,
		providerName: providerName,
		authManager:  authMgr,
		config:       cfg,
	}
}

func (p *authManagedProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	ctx, cred := p.pickCredential(ctx, req.Model)
	resp, err := p.base.Generate(ctx, req)
	p.markResult(ctx, cred, req.Model, err)
	return resp, err
}

func (p *authManagedProvider) Stream(ctx context.Context, req *GenerateRequest) (*schema.StreamReader[*schema.Message], error) {
	ctx, cred := p.pickCredential(ctx, req.Model)
	stream, err := p.base.Stream(ctx, req)
	p.markResult(ctx, cred, req.Model, err)
	return stream, err
}

func (p *authManagedProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return p.base.ListModels(ctx)
}

func (p *authManagedProvider) Capabilities() ProviderCapabilities {
	return p.base.Capabilities()
}

func (p *authManagedProvider) Config() ProviderConfig {
	return p.config
}

func (p *authManagedProvider) pickCredential(ctx context.Context, model string) (context.Context, *credential.Credential) {
	if p.authManager == nil || !p.shouldUseCredential() {
		return ctx, nil
	}
	cred, err := p.authManager.Pick(ctx, p.providerName, model, nil)
	if err != nil || cred == nil {
		return ctx, nil
	}
	return WithCredential(ctx, cred), cred
}

func (p *authManagedProvider) markResult(ctx context.Context, cred *credential.Credential, model string, err error) {
	if p.authManager == nil || cred == nil {
		return
	}

	result := manager.Result{
		CredentialID: cred.ID,
		Provider:     cred.Provider,
		Model:        model,
		Success:      err == nil,
	}
	if err != nil {
		var se StatusError
		httpStatus := http.StatusBadGateway
		if errors.As(err, &se) {
			httpStatus = se.StatusCode()
		}
		result.Error = &credential.Error{
			Code:       http.StatusText(httpStatus),
			Message:    err.Error(),
			HTTPStatus: httpStatus,
			Retryable:  httpStatus == http.StatusTooManyRequests || httpStatus >= 500,
		}
	}
	p.authManager.MarkResult(ctx, result)
}

func (p *authManagedProvider) shouldUseCredential() bool {
	switch p.config.AuthStrategy {
	case AuthStrategyAPIKeyOnly:
		return false
	case AuthStrategyAPIKeyFirst:
		return strings.TrimSpace(p.config.APIKey) == ""
	case AuthStrategyCredentialFirst, AuthStrategyCredentialOnly:
		return true
	default:
		return strings.TrimSpace(p.config.APIKey) == ""
	}
}
