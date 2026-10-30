# Wabo

Wabo is a Go SDK for the WhatsApp Business Platform: messaging, business
management, templates, media, Flows and webhook events.

This SDK is in beta and is not affiliated with Meta or WhatsApp.
It requires Go 1.26 or newer; go.mod specifies the development toolchain.

## Installation

```sh
go get github.com/gTahidi/wabo@v0.10.0-beta.1
```

For migration from Wapi, see [the migration guide](docs/guide/migrating-to-wabo.md).

## Usage

```go
package main

import (
    "os"

    "github.com/gTahidi/wabo"
    "github.com/gTahidi/wabo/pkg/components"
)

func main() {
    client := wabo.New(&wabo.ClientConfig{
        BusinessAccountId: os.Getenv("WHATSAPP_BUSINESS_ACCOUNT_ID"),
        ApiAccessToken: os.Getenv("WHATSAPP_ACCESS_TOKEN"),
        WebhookSecret: os.Getenv("WHATSAPP_VERIFY_TOKEN"),
        AppSecret: os.Getenv("META_APP_SECRET"),
    })
    messaging := client.NewMessagingClient(os.Getenv("WHATSAPP_PHONE_NUMBER_ID"))
    message, err := components.NewTextMessage(components.TextMessageConfigs{Text: "Hello from Wabo"})
    if err != nil {
        panic(err)
    }
    _, err = messaging.Message.Send(message, os.Getenv("WHATSAPP_RECIPIENT"))
    if err != nil {
        panic(err)
    }
}
```

Use `ClientConfig.APIVersion` to select a Graph version explicitly. Empty retains
`v24.0`; selecting a newer version does not certify all API features against it.

Examples: [chat bot](examples/chat-bot) and
[HTTP backend integration](examples/http-backend-integration).
Public service packages remain under `pkg/`; orchestration helpers are in `manager/`.

## Development

```sh
go build ./...
go test ./...
make format
make docs
```

See [repository guidelines](AGENTS.md) and [commit conventions](COMMIT_CONVENTION.md).

Distributed under the [Apache 2.0 license](LICENSE).
