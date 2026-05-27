package internal

import (
	"context"
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	kratos "github.com/ory/kratos-client-go"
)

type stepConstructor func(name string, config map[string]any) (sdk.StepInstance, error)

var stepRegistry = map[string]stepConstructor{
	"step.ory_kratos_auth_provider_describe":   newAuthProviderDescribeStep,
	"step.ory_kratos_identity_create":          newKratosStep(oryKratosIdentityCreate),
	"step.ory_kratos_identity_get":             newKratosStep(oryKratosIdentityGet),
	"step.ory_kratos_identity_list":            newKratosStep(oryKratosIdentityList),
	"step.ory_kratos_identity_update":          newKratosStep(oryKratosIdentityUpdate),
	"step.ory_kratos_identity_delete":          newKratosStep(oryKratosIdentityDelete),
	"step.ory_kratos_login_flow_create":        newKratosStep(oryKratosLoginFlowCreate),
	"step.ory_kratos_login_flow_get":           newKratosStep(oryKratosLoginFlowGet),
	"step.ory_kratos_registration_flow_create": newKratosStep(oryKratosRegistrationFlowCreate),
	"step.ory_kratos_registration_flow_get":    newKratosStep(oryKratosRegistrationFlowGet),
	"step.ory_kratos_recovery_flow_create":     newKratosStep(oryKratosRecoveryFlowCreate),
	"step.ory_kratos_recovery_flow_get":        newKratosStep(oryKratosRecoveryFlowGet),
	"step.ory_kratos_verification_flow_create": newKratosStep(oryKratosVerificationFlowCreate),
	"step.ory_kratos_verification_flow_get":    newKratosStep(oryKratosVerificationFlowGet),
	"step.ory_kratos_identity_recovery_code":   newKratosStep(oryKratosIdentityRecoveryCode),
	"step.ory_kratos_identity_sessions_delete": newKratosStep(oryKratosIdentitySessionsDelete),
}

func allStepTypes() []string {
	return []string{
		"step.ory_kratos_auth_provider_describe",
		"step.ory_kratos_identity_create",
		"step.ory_kratos_identity_get",
		"step.ory_kratos_identity_list",
		"step.ory_kratos_identity_update",
		"step.ory_kratos_identity_delete",
		"step.ory_kratos_login_flow_create",
		"step.ory_kratos_login_flow_get",
		"step.ory_kratos_registration_flow_create",
		"step.ory_kratos_registration_flow_get",
		"step.ory_kratos_recovery_flow_create",
		"step.ory_kratos_recovery_flow_get",
		"step.ory_kratos_verification_flow_create",
		"step.ory_kratos_verification_flow_get",
		"step.ory_kratos_identity_recovery_code",
		"step.ory_kratos_identity_sessions_delete",
	}
}

func createStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	constructor, ok := stepRegistry[typeName]
	if !ok {
		return nil, fmt.Errorf("ory kratos plugin: unknown step type %q", typeName)
	}
	return constructor(name, config)
}

type kratosHandler func(context.Context, *OryKratosClient, map[string]any) (map[string]any, error)

type kratosStep struct {
	name       string
	moduleName string
	handler    kratosHandler
}

func newKratosStep(handler kratosHandler) stepConstructor {
	return func(name string, config map[string]any) (sdk.StepInstance, error) {
		moduleName := stringValue(config, "module")
		if moduleName == "" {
			moduleName = "ory_kratos"
		}
		return &kratosStep{name: name, moduleName: moduleName, handler: handler}, nil
	}
}

func (s *kratosStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _, config map[string]any) (*sdk.StepResult, error) {
	client, ok := GetClient(s.moduleName)
	if !ok {
		return &sdk.StepResult{Output: map[string]any{"error": "ory kratos client not found: " + s.moduleName}}, nil
	}
	output, err := s.handler(ctx, client, mergeMaps(config, current))
	if err != nil {
		return &sdk.StepResult{Output: errResult(err)}, nil
	}
	return &sdk.StepResult{Output: output}, nil
}

func oryKratosIdentityCreate(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	body, err := createIdentityBody(values)
	if err != nil {
		return nil, err
	}
	identity, _, err := client.Admin.IdentityAPI.CreateIdentity(ctx).CreateIdentityBody(body).Execute()
	if err != nil {
		return nil, err
	}
	return map[string]any{"identity": identityToMap(identity)}, nil
}

func oryKratosIdentityGet(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "identity_id", "identityId", "id")
	if id == "" {
		return nil, fmt.Errorf("identity_id is required")
	}
	identity, _, err := client.Admin.IdentityAPI.GetIdentity(ctx, id).Execute()
	if err != nil {
		return nil, err
	}
	return map[string]any{"identity": identityToMap(identity)}, nil
}

func oryKratosIdentityList(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	req := client.Admin.IdentityAPI.ListIdentities(ctx)
	if page := intValue(values, "page", 0); page > 0 {
		req = req.Page(int64(page))
	}
	if perPage := intValue(values, "per_page", 0); perPage > 0 {
		req = req.PerPage(int64(perPage))
	}
	if identifier := stringValue(values, "credentials_identifier"); identifier != "" {
		req = req.CredentialsIdentifier(identifier)
	}
	identities, _, err := req.Execute()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(identities))
	for i := range identities {
		out = append(out, identityToMap(&identities[i]))
	}
	return map[string]any{"identities": out}, nil
}

func oryKratosIdentityUpdate(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "identity_id", "identityId", "id")
	if id == "" {
		return nil, fmt.Errorf("identity_id is required")
	}
	body, err := updateIdentityBody(values)
	if err != nil {
		return nil, err
	}
	identity, _, err := client.Admin.IdentityAPI.UpdateIdentity(ctx, id).UpdateIdentityBody(body).Execute()
	if err != nil {
		return nil, err
	}
	return map[string]any{"updated": true, "identity": identityToMap(identity)}, nil
}

func oryKratosIdentityDelete(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "identity_id", "identityId", "id")
	if id == "" {
		return nil, fmt.Errorf("identity_id is required")
	}
	if _, err := client.Admin.IdentityAPI.DeleteIdentity(ctx, id).Execute(); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "identity_id": id}, nil
}

func oryKratosLoginFlowCreate(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	req := client.Public.FrontendAPI.CreateNativeLoginFlow(ctx)
	if aal := stringValue(values, "aal"); aal != "" {
		req = req.Aal(aal)
	}
	if organization := stringValue(values, "organization"); organization != "" {
		req = req.Organization(organization)
	}
	flow, _, err := req.Execute()
	return encodedFlow("login_flow", flow, err)
}

func oryKratosLoginFlowGet(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	id, err := flowID(values)
	if err != nil {
		return nil, err
	}
	flow, _, err := client.Public.FrontendAPI.GetLoginFlow(ctx).Id(id).Execute()
	return encodedFlow("login_flow", flow, err)
}

func oryKratosRegistrationFlowCreate(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	req := client.Public.FrontendAPI.CreateNativeRegistrationFlow(ctx)
	if organization := stringValue(values, "organization"); organization != "" {
		req = req.Organization(organization)
	}
	flow, _, err := req.Execute()
	return encodedFlow("registration_flow", flow, err)
}

func oryKratosRegistrationFlowGet(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	id, err := flowID(values)
	if err != nil {
		return nil, err
	}
	flow, _, err := client.Public.FrontendAPI.GetRegistrationFlow(ctx).Id(id).Execute()
	return encodedFlow("registration_flow", flow, err)
}

func oryKratosRecoveryFlowCreate(ctx context.Context, client *OryKratosClient, _ map[string]any) (map[string]any, error) {
	flow, _, err := client.Public.FrontendAPI.CreateNativeRecoveryFlow(ctx).Execute()
	return encodedFlow("recovery_flow", flow, err)
}

func oryKratosRecoveryFlowGet(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	id, err := flowID(values)
	if err != nil {
		return nil, err
	}
	flow, _, err := client.Public.FrontendAPI.GetRecoveryFlow(ctx).Id(id).Execute()
	return encodedFlow("recovery_flow", flow, err)
}

func oryKratosVerificationFlowCreate(ctx context.Context, client *OryKratosClient, _ map[string]any) (map[string]any, error) {
	flow, _, err := client.Public.FrontendAPI.CreateNativeVerificationFlow(ctx).Execute()
	return encodedFlow("verification_flow", flow, err)
}

func oryKratosVerificationFlowGet(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	id, err := flowID(values)
	if err != nil {
		return nil, err
	}
	flow, _, err := client.Public.FrontendAPI.GetVerificationFlow(ctx).Id(id).Execute()
	return encodedFlow("verification_flow", flow, err)
}

func oryKratosIdentityRecoveryCode(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "identity_id", "identityId", "id")
	if id == "" {
		return nil, fmt.Errorf("identity_id is required")
	}
	body := kratos.NewCreateRecoveryCodeForIdentityBody(id)
	code, _, err := client.Admin.IdentityAPI.CreateRecoveryCodeForIdentity(ctx).CreateRecoveryCodeForIdentityBody(*body).Execute()
	return encodedFlow("recovery_code", code, err)
}

func oryKratosIdentitySessionsDelete(ctx context.Context, client *OryKratosClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "identity_id", "identityId", "id")
	if id == "" {
		return nil, fmt.Errorf("identity_id is required")
	}
	if _, err := client.Admin.IdentityAPI.DeleteIdentitySessions(ctx, id).Execute(); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "identity_id": id}, nil
}

func createIdentityBody(values map[string]any) (kratos.CreateIdentityBody, error) {
	schemaID := firstNonEmpty(values, "schema_id", "schemaId")
	if schemaID == "" {
		return kratos.CreateIdentityBody{}, fmt.Errorf("schema_id is required")
	}
	traits := mapValue(values, "traits")
	if traits == nil {
		return kratos.CreateIdentityBody{}, fmt.Errorf("traits is required")
	}
	body := kratos.NewCreateIdentityBody(schemaID, traits)
	if state := stringValue(values, "state"); state != "" {
		body.SetState(state)
	}
	if metadata := mapValue(values, "metadata_admin"); metadata != nil {
		body.SetMetadataAdmin(metadata)
	}
	if metadata := mapValue(values, "metadata_public"); metadata != nil {
		body.SetMetadataPublic(metadata)
	}
	return *body, nil
}

func updateIdentityBody(values map[string]any) (kratos.UpdateIdentityBody, error) {
	source := values
	if payload := mapValue(values, "identity"); payload != nil {
		source = payload
	}
	schemaID := firstNonEmpty(source, "schema_id", "schemaId")
	if schemaID == "" {
		return kratos.UpdateIdentityBody{}, fmt.Errorf("schema_id is required")
	}
	traits := mapValue(source, "traits")
	if traits == nil {
		return kratos.UpdateIdentityBody{}, fmt.Errorf("traits is required")
	}
	state := stringValue(source, "state")
	if state == "" {
		state = "active"
	}
	body := kratos.NewUpdateIdentityBody(schemaID, state, traits)
	if metadata := mapValue(source, "metadata_admin"); metadata != nil {
		body.SetMetadataAdmin(metadata)
	}
	if metadata := mapValue(source, "metadata_public"); metadata != nil {
		body.SetMetadataPublic(metadata)
	}
	return *body, nil
}

func identityToMap(identity *kratos.Identity) map[string]any {
	if identity == nil {
		return nil
	}
	out := map[string]any{
		"id":              identity.Id,
		"schema_id":       identity.SchemaId,
		"schema_url":      identity.SchemaUrl,
		"state":           stringPtrValue(identity.State),
		"traits":          identity.Traits,
		"metadata_admin":  identity.MetadataAdmin,
		"metadata_public": identity.MetadataPublic,
	}
	return compactMap(out)
}

func encodedFlow(key string, value any, err error) (map[string]any, error) {
	if err != nil {
		return nil, err
	}
	encoded, err := encodeValue(value)
	if err != nil {
		return nil, err
	}
	return map[string]any{key: encoded}, nil
}

func flowID(values map[string]any) (string, error) {
	id := firstNonEmpty(values, "flow_id", "flowId", "id")
	if id == "" {
		return "", fmt.Errorf("flow_id is required")
	}
	return id, nil
}

func firstNonEmpty(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values, key); value != "" {
			return value
		}
	}
	return ""
}
