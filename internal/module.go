package internal

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	kratos "github.com/ory/kratos-client-go"
)

type oryKratosModule struct {
	name   string
	config map[string]any
}

func newOryKratosModule(name string, config map[string]any) (*oryKratosModule, error) {
	return &oryKratosModule{name: name, config: config}, nil
}

func (m *oryKratosModule) Init() error {
	adminURL := firstNonEmpty(m.config, "admin_url", "adminUrl", "admin")
	if adminURL == "" {
		return fmt.Errorf("ory.kratos %q: admin_url is required", m.name)
	}
	publicURL := firstNonEmpty(m.config, "public_url", "publicUrl", "public")
	if publicURL == "" {
		publicURL = adminURL
	}
	apiKey := firstNonEmpty(m.config, "api_key", "apiKey", "token")

	adminClient, err := newKratosClient(adminURL, apiKey)
	if err != nil {
		return fmt.Errorf("ory.kratos %q: create admin client: %w", m.name, err)
	}
	publicClient, err := newKratosClient(publicURL, "")
	if err != nil {
		return fmt.Errorf("ory.kratos %q: create public client: %w", m.name, err)
	}
	RegisterClient(m.name, &OryKratosClient{Admin: adminClient, Public: publicClient, AdminURL: strings.TrimRight(adminURL, "/"), PublicURL: strings.TrimRight(publicURL, "/")})
	return nil
}

func (m *oryKratosModule) Start(context.Context) error { return nil }

func (m *oryKratosModule) Stop(context.Context) error {
	UnregisterClient(m.name)
	return nil
}

func newKratosClient(rawURL, apiKey string) (*kratos.APIClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("url must include scheme and host")
	}
	cfg := kratos.NewConfiguration()
	cfg.Scheme = parsed.Scheme
	cfg.Host = parsed.Host
	if path := strings.TrimRight(parsed.Path, "/"); path != "" {
		cfg.Servers = kratos.ServerConfigurations{{URL: parsed.Scheme + "://" + parsed.Host + path}}
	}
	if apiKey != "" {
		cfg.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	}
	cfg.UserAgent = "workflow-plugin-ory-kratos/" + Version
	return kratos.NewAPIClient(cfg), nil
}
