package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-ory-kratos/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestPluginManifestAdvertisesRequiredSecrets(t *testing.T) {
	raw, err := os.ReadFile("../plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		RequiredSecrets []struct {
			Name      string `json:"name"`
			Sensitive bool   `json:"sensitive"`
		} `json:"required_secrets"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	secrets := map[string]bool{}
	for _, secret := range manifest.RequiredSecrets {
		secrets[secret.Name] = secret.Sensitive
	}
	for name, sensitive := range map[string]bool{
		"ORY_KRATOS_ADMIN_URL":  false,
		"ORY_KRATOS_PUBLIC_URL": false,
		"ORY_KRATOS_API_KEY":    true,
	} {
		got, ok := secrets[name]
		if !ok {
			t.Fatalf("plugin.json missing required_secrets entry %s", name)
		}
		if got != sensitive {
			t.Fatalf("%s sensitive = %v, want %v", name, got, sensitive)
		}
	}
}

func TestModuleInitRegistersKratosClients(t *testing.T) {
	module, err := newOryKratosModule("ory_kratos-test", map[string]any{
		"adminUrl":  "https://kratos-admin.example.test",
		"publicUrl": "https://kratos-public.example.test",
		"apiKey":    "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	client, ok := GetClient("ory_kratos-test")
	if !ok || client == nil {
		t.Fatal("expected registered client")
	}
	if client.Admin == nil || client.Public == nil {
		t.Fatal("expected kratos clients")
	}
	if client.AdminURL != "https://kratos-admin.example.test" {
		t.Fatalf("admin url = %q", client.AdminURL)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetClient("ory_kratos-test"); ok {
		t.Fatal("expected client to be unregistered")
	}
}

func TestModuleInitRequiresAdminURL(t *testing.T) {
	module, err := newOryKratosModule("ory_kratos-test", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err == nil {
		t.Fatal("expected missing admin url error")
	}
}

func TestContractRegistryIncludesStrictProtoDescriptors(t *testing.T) {
	provider, ok := NewOryKratosPlugin().(interface {
		ContractRegistry() *pb.ContractRegistry
	})
	if !ok {
		t.Fatal("plugin does not expose ContractRegistry")
	}
	registry := provider.ContractRegistry()
	if registry == nil || registry.GetFileDescriptorSet() == nil {
		t.Fatal("missing contract registry file descriptors")
	}
	contractsByType := map[string]*pb.ContractDescriptor{}
	for _, contract := range registry.GetContracts() {
		switch contract.GetKind() {
		case pb.ContractKind_CONTRACT_KIND_MODULE:
			contractsByType["module:"+contract.GetModuleType()] = contract
		case pb.ContractKind_CONTRACT_KIND_STEP:
			contractsByType["step:"+contract.GetStepType()] = contract
		}
	}
	module := contractsByType["module:ory.kratos"]
	if module == nil || module.GetConfigMessage() != "workflow.plugins.ory_kratos.v1.ProviderConfig" {
		t.Fatalf("unexpected module contract: %#v", module)
	}
	for _, stepType := range allStepTypes() {
		contract := contractsByType["step:"+stepType]
		if contract == nil {
			t.Fatalf("missing step contract %s", stepType)
		}
		if contract.GetMode() != pb.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %v", stepType, contract.GetMode())
		}
	}
}

func TestDescriptorAdvertisesBackedCapabilities(t *testing.T) {
	step, err := newAuthProviderDescribeStep("describe", map[string]any{"adminUrl": "https://kratos-admin.example.test", "publicUrl": "https://kratos-public.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	providers := result.Output["providers"].([]map[string]any)
	if len(providers) != 1 {
		t.Fatalf("providers = %#v", providers)
	}
	provider := providers[0]
	categories := stringSet(provider["categories"].([]string))
	for _, category := range []string{"identity_management", "authentication_method"} {
		if !categories[category] {
			t.Fatalf("missing category %q", category)
		}
	}
	if categories["oauth2_oidc"] {
		t.Fatal("descriptor must not advertise OAuth/OIDC provider support; Hydra owns that category")
	}
	if categories["enterprise_sso"] {
		t.Fatal("descriptor must not advertise enterprise SSO; Polis/SSO providers own that category")
	}
	if categories["rbac"] {
		t.Fatal("descriptor must not advertise RBAC role management without role mutation steps")
	}
	if categories["mfa"] {
		t.Fatal("descriptor must not advertise MFA management without OryKratos MFA steps")
	}
	if categories["scim"] {
		t.Fatal("descriptor must not advertise SCIM without provisioning steps")
	}
	capabilities := provider["capabilities"].([]map[string]any)
	if len(capabilities) != 2 {
		t.Fatalf("capability count = %d, want 2", len(capabilities))
	}
	for _, capability := range capabilities {
		if capability["supported"] != true {
			t.Fatalf("%s supported = %#v", capability["key"], capability["supported"])
		}
	}
}

func TestTypedDescriptor(t *testing.T) {
	result, err := typedAuthProviderDescribe(context.Background(), sdk.TypedStepRequest[*contracts.AuthProviderDescribeConfig, *contracts.AuthProviderDescribeInput]{
		Config: &contracts.AuthProviderDescribeConfig{ProviderId: "ory_kratos-admin"},
		Input:  &contracts.AuthProviderDescribeInput{AdminUrl: "https://kratos-admin.example.test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output == nil || len(result.Output.GetProviders()) != 1 {
		t.Fatalf("providers = %#v", result.Output)
	}
	if result.Output.GetProviders()[0].GetId() != "ory_kratos-admin" {
		t.Fatalf("provider id = %q", result.Output.GetProviders()[0].GetId())
	}
}

func TestMissingClientReturnsStepError(t *testing.T) {
	step, err := createStep("step.ory_kratos_identity_get", "get", map[string]any{"module": "missing"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"identity_id": "123"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["error"] == nil {
		t.Fatalf("expected error output, got %#v", result.Output)
	}
}

func TestIdentityCreateUsesKratosAdminAPI(t *testing.T) {
	var gotPath string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if gotPath != "/admin/identities" {
			t.Fatalf("path = %s", gotPath)
		}
		body := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["schema_id"] != "default" {
			t.Fatalf("schema_id = %#v", body["schema_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"id-1","schema_id":"default","schema_url":"https://schemas.example/default","state":"active","traits":{"email":"user@example.com"}}`))
	}))
	defer server.Close()

	module, err := newOryKratosModule("ory_kratos-test", map[string]any{"adminUrl": server.URL, "publicUrl": server.URL, "apiKey": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	defer module.Stop(context.Background())

	step, err := createStep("step.ory_kratos_identity_create", "create", map[string]any{"module": "ory_kratos-test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"schema_id": "default", "traits": map[string]any{"email": "user@example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["identity"].(map[string]any)["id"] != "id-1" {
		t.Fatalf("output = %#v", result.Output)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("authorization header = %q", gotAuth)
	}
}

func TestLoginFlowCreateUsesKratosPublicAPI(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if gotPath != "/self-service/login/api" {
			t.Fatalf("path = %s", gotPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"flow-1","type":"api","expires_at":"2026-05-27T18:00:00Z","issued_at":"2026-05-27T17:00:00Z","request_url":"https://kratos.example/self-service/login/api","state":"choose_method","ui":{"action":"https://kratos.example/self-service/login?flow=flow-1","method":"POST","nodes":[]}}`))
	}))
	defer server.Close()

	module, err := newOryKratosModule("ory_kratos-test", map[string]any{"adminUrl": server.URL, "publicUrl": server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	defer module.Stop(context.Background())

	step, err := createStep("step.ory_kratos_login_flow_create", "login", map[string]any{"module": "ory_kratos-test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["login_flow"].(map[string]any)["id"] != "flow-1" {
		t.Fatalf("output = %#v", result.Output)
	}
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
