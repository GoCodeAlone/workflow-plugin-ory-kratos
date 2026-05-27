package internal

import (
	"context"
	"strings"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

type authProviderDescribeStep struct {
	name   string
	config map[string]any
}

func newAuthProviderDescribeStep(name string, config map[string]any) (sdk.StepInstance, error) {
	return &authProviderDescribeStep{name: name, config: config}, nil
}

func (s *authProviderDescribeStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, current, _, _ map[string]any) (*sdk.StepResult, error) {
	values := mergeMaps(s.config, current)
	providerID := firstNonEmpty(values, "provider_id", "providerId")
	if providerID == "" {
		providerID = "ory_kratos"
	}
	adminURL := firstNonEmpty(values, "admin_url", "adminUrl")
	publicURL := firstNonEmpty(values, "public_url", "publicUrl")
	return &sdk.StepResult{Output: map[string]any{
		"providers": []map[string]any{oryKratosProviderDescriptor(providerID, adminURL, publicURL)},
	}}, nil
}

func oryKratosProviderDescriptor(providerID, adminURL, publicURL string) map[string]any {
	return map[string]any{
		"id":             providerID,
		"label":          "Ory Kratos",
		"description":    "Ory Kratos identity administration and self-service authentication flow integration.",
		"categories":     []string{"identity_management", "authentication_method"},
		"implementation": "workflow-plugin-ory-kratos",
		"version":        Version,
		"docs_url":       "https://github.com/GoCodeAlone/workflow-plugin-ory-kratos",
		"support_level":  "management",
		"capabilities": []map[string]any{
			oryKratosCapability("ory_kratos_identities", "Identities", "identity_management", "List, read, create, update, and delete Kratos identities through the Admin API.", []string{"kratos.identities.read", "kratos.identities.write"}, oryKratosFields(adminURL, publicURL)),
			oryKratosCapability("ory_kratos_self_service_flows", "Self-service flows", "authentication_method", "Create and inspect native login, registration, recovery, and verification flows through the Public API.", []string{"kratos.flows.read", "kratos.flows.write"}, oryKratosFields(adminURL, publicURL)),
		},
	}
}

func oryKratosCapability(key, label, category, description string, appScopes []string, fields []map[string]any) map[string]any {
	return map[string]any{
		"key":                key,
		"label":              label,
		"category":           category,
		"description":        description,
		"supported":          true,
		"app_scopes":         appScopes,
		"admin_read_scopes":  []string{"admin.auth.providers.read"},
		"admin_write_scopes": []string{"admin.auth.providers.write"},
		"config_fields":      fields,
	}
}

func oryKratosFields(adminURL, publicURL string) []map[string]any {
	return []map[string]any{
		oryKratosField("ory_kratos_admin_url", "Admin API URL", "url", "Base URL for the Kratos Admin API.", "Keep this private. Do not expose the Admin API directly to browsers.", false, true, optionIfSet(strings.TrimRight(adminURL, "/"))),
		oryKratosField("ory_kratos_public_url", "Public API URL", "url", "Base URL for the Kratos Public API.", "Use the browser-reachable Kratos public endpoint for self-service flows.", false, true, optionIfSet(strings.TrimRight(publicURL, "/"))),
		oryKratosField("ory_kratos_api_key", "Admin API bearer token", "secret", "Optional bearer token for protected Admin API deployments.", "Write-only secret. Prefer network isolation and least-privilege reverse-proxy credentials.", true, false, nil),
	}
}

func oryKratosField(key, label, inputType, description, helpText string, secret, required bool, options []map[string]any) map[string]any {
	return map[string]any{
		"key":         key,
		"label":       label,
		"input_type":  inputType,
		"description": description,
		"help_text":   helpText,
		"secret":      secret,
		"required":    required,
		"options":     options,
	}
}

func optionIfSet(value string) []map[string]any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []map[string]any{{"value": value, "label": value}}
}
