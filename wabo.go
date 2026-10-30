// Package wabo provides a WhatsApp Business Platform SDK.
// Create a Client with New, then use its business and messaging services.
package wabo

import client "github.com/gTahidi/wabo/pkg/client"

// Client manages business, messaging, media, Flow and webhook services.
type Client = client.Client

// ClientConfig configures credentials, transport and webhook handling.
type ClientConfig = client.ClientConfig

// New creates a client with the supplied configuration.
func New(config *ClientConfig) *Client {
	return client.New(config)
}
