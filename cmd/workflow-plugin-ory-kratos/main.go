package main

import (
	"github.com/GoCodeAlone/workflow-plugin-ory-kratos/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewOryKratosPlugin(), sdk.WithBuildVersion(sdk.ResolveBuildVersion(internal.Version)))
}
