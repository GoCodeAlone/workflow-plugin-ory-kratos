# workflow-plugin-ory-kratos

Ory Kratos identity provider plugin for Workflow. It uses the official
`github.com/ory/kratos-client-go` SDK.

## Capabilities

- `ory.kratos` module using Kratos Admin and Public API base URLs
- Auth provider descriptor step for admin catalog integration
- Identity create/read/list/update/delete steps
- Native login, registration, recovery, and verification flow create/get steps
- Identity recovery-code creation and session deletion steps

The descriptor advertises only capabilities backed by the plugin's concrete
management steps.

## Install

```sh
wfctl plugin install workflow-plugin-ory-kratos
```

## License

MIT
