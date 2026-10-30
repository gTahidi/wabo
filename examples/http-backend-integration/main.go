package main

import (
	"fmt"

	"github.com/gTahidi/wabo"
	"github.com/gTahidi/wabo/pkg/components"
	"github.com/gTahidi/wabo/pkg/events"
	"github.com/labstack/echo/v4"
)

func main() {

	client := wabo.New(&wabo.ClientConfig{
		ApiAccessToken:    "",
		BusinessAccountId: "",
		WebhookPath:       "/webhook",
		WebhookSecret:     "1234567890",
		WebhookServerPort: 8080,
	})

	client.On(events.TextMessageEventType, func(event events.BaseEvent) {
		textMessageEvent := event.(*events.TextMessageEvent)
		reply, err := components.NewTextMessage(components.TextMessageConfigs{
			Text: "Hello, from wabo",
		})
		if err != nil {
			fmt.Println("error creating text message", err)
			return
		}
		textMessageEvent.Reply(reply)
	})

	getHandler := client.GetWebhookGetRequestHandler()
	postHandler := client.GetWebhookPostRequestHandler()

	server := echo.New()

	server.GET("/webhook", getHandler)
	server.POST("/webhook", postHandler)

	server.Start(":8080")

}
