package internal

import (
	"fmt"

	"github.com/GoCodeAlone/workflow-plugin-ory-kratos/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

var Version = "0.0.0"

type ory_kratosPlugin struct{}

func NewOryKratosPlugin() sdk.PluginProvider {
	return &ory_kratosPlugin{}
}

func (p *ory_kratosPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-ory-kratos",
		Version:     Version,
		Author:      "GoCodeAlone",
		Description: "Ory Kratos identity provider plugin backed by the official Kratos Go SDK",
	}
}

func (p *ory_kratosPlugin) ModuleTypes() []string {
	return []string{"ory.kratos"}
}

func (p *ory_kratosPlugin) TypedModuleTypes() []string {
	return p.ModuleTypes()
}

func (p *ory_kratosPlugin) CreateModule(typeName, name string, config map[string]any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "ory.kratos":
		return newOryKratosModule(name, config)
	default:
		return nil, fmt.Errorf("ory_kratos plugin: unknown module type %q", typeName)
	}
}

func (p *ory_kratosPlugin) CreateTypedModule(typeName, name string, config *anypb.Any) (sdk.ModuleInstance, error) {
	if typeName != "ory.kratos" {
		return nil, fmt.Errorf("ory_kratos plugin: unknown typed module type %q", typeName)
	}
	factory := sdk.NewTypedModuleFactory(typeName, &contracts.ProviderConfig{}, func(name string, cfg *contracts.ProviderConfig) (sdk.ModuleInstance, error) {
		return newOryKratosModule(name, typedModuleConfig(cfg))
	})
	return factory.CreateTypedModule(typeName, name, config)
}

func (p *ory_kratosPlugin) StepTypes() []string {
	return allStepTypes()
}

func (p *ory_kratosPlugin) TypedStepTypes() []string {
	return p.StepTypes()
}

func (p *ory_kratosPlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	return createStep(typeName, name, config)
}

func (p *ory_kratosPlugin) CreateTypedStep(typeName, name string, config *anypb.Any) (sdk.StepInstance, error) {
	if _, ok := stepRegistry[typeName]; !ok {
		return nil, fmt.Errorf("%w: step type %q", sdk.ErrTypedContractNotHandled, typeName)
	}
	if typeName == "step.ory_kratos_auth_provider_describe" {
		return sdk.NewTypedStepFactory(typeName, &contracts.AuthProviderDescribeConfig{}, &contracts.AuthProviderDescribeInput{}, typedAuthProviderDescribe).CreateTypedStep(typeName, name, config)
	}
	return sdk.NewTypedStepFactory(typeName, &contracts.OryKratosStepConfig{}, &contracts.OryKratosStepInput{}, typedStepHandler(typeName)).CreateTypedStep(typeName, name, config)
}

func (p *ory_kratosPlugin) ContractRegistry() *pb.ContractRegistry {
	return contractRegistry
}

var contractRegistry = &pb.ContractRegistry{
	FileDescriptorSet: &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto),
			protodesc.ToFileDescriptorProto(contracts.File_internal_contracts_ory_kratos_proto),
		},
	},
	Contracts: contractDescriptors(),
}

func contractDescriptors() []*pb.ContractDescriptor {
	descriptors := []*pb.ContractDescriptor{
		moduleContract("ory.kratos", "ProviderConfig"),
	}
	for _, stepType := range allStepTypes() {
		if stepType == "step.ory_kratos_auth_provider_describe" {
			descriptors = append(descriptors, stepContract(stepType, "AuthProviderDescribeConfig", "AuthProviderDescribeInput", "AuthProviderDescribeOutput"))
			continue
		}
		descriptors = append(descriptors, stepContract(stepType, "OryKratosStepConfig", "OryKratosStepInput", "OryKratosStepOutput"))
	}
	return descriptors
}

func moduleContract(moduleType, configMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.ory_kratos.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_MODULE,
		ModuleType:    moduleType,
		ConfigMessage: pkg + configMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}

func stepContract(stepType, configMessage, inputMessage, outputMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.ory_kratos.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_STEP,
		StepType:      stepType,
		ConfigMessage: pkg + configMessage,
		InputMessage:  pkg + inputMessage,
		OutputMessage: pkg + outputMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}
