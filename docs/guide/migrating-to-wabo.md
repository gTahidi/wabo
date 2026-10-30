# Migrating from Wapi to Wabo

The module is now `github.com/gTahidi/wabo`, and the SDK's entry-point package
is named `wabo`. This is a breaking import-path migration. Existing Wapi releases
remain separate; this module does not transparently replace them.

## Application changes

Replace `github.com/gTahidi/wapi.go` with `github.com/gTahidi/wabo` in every Go
import. For the client entry point, prefer the new root import:

```go
import "github.com/gTahidi/wabo"

client := wabo.New(&wabo.ClientConfig{
    BusinessAccountId: businessID,
    ApiAccessToken: accessToken,
    WebhookSecret: verifyToken,
})
```

Replace `wapi.` references with `wabo.`. The existing `pkg/client` directory
remains available under the new module, also with package name `wabo`.
Other directories retain their names, for example `pkg/components`, `pkg/events`,
`pkg/business`, `pkg/messaging` and `manager`.

Run `go get github.com/gTahidi/wabo@v0.10.0-beta.1`, then `go mod tidy`
and your application tests. Do not mix old and new module imports: Go treats their
SDK types as distinct. Update dependencies that expose SDK types together.

For local evaluation before publication, use a temporary application go.mod entry:

```text
require github.com/gTahidi/wabo v0.0.0
replace github.com/gTahidi/wabo => /absolute/path/to/this/checkout
```

Remove the local replacement when installing a published version.

## Behavior carried forward

The previous correctness changes are included. Incoming contact cards are exposed
through `ContactCards`; decoded webhook `Change.Value` contains `json.RawMessage`.
See [the implementation report](improvement-execution.md) for details and the
remaining API verification work. The rename does not change the Graph API default
or establish complete BSUID support.

## Release

The repository is `gTahidi/wabo`; the migration release is `v0.10.0-beta.1`.
The old repository redirects, but consumers must still migrate their imports.
Releases are published explicitly; the semantic-release workflow is available
through manual dispatch rather than publishing automatically on every master push.
