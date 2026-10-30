package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gTahidi/wabo"
	"github.com/gTahidi/wabo/pkg/business"
	waboComponents "github.com/gTahidi/wabo/pkg/components"
	"github.com/gTahidi/wabo/pkg/events"
)

func main() {
	// creating a client
	businessAccountId := ""
	phoneNumber := ""

	client := wabo.New(&wabo.ClientConfig{
		ApiAccessToken:    "",
		BusinessAccountId: businessAccountId,
		WebhookPath:       "/webhook",
		WebhookSecret:     "1234567890",
		WebhookServerPort: 8080,
	})

	// messaging client is specific to a phone number, if in case you are looking to change the mobile the number you need to create a new messaging client
	// messagingClient := client.NewMessagingClient("113269274970227")

	// uncomment to try this business api endpoint to get the conversation analytics
	// client.Business.ConversationAnalytics(business.ConversationAnalyticsOptions{
	// 	Start:       time.Now().Add(-time.Hour * 24 * 7 * 30),
	// 	End:         time.Now(),
	// 	Granularity: business.ConversationAnalyticsGranularityTypeDay,
	// })

	client.Business.FetchAnalytics(business.AccountAnalyticsOptions{
		Start:       time.Now().Add(-time.Hour * 24 * 7 * 30),
		End:         time.Now(),
		Granularity: business.AnalyticsRequestGranularityTypeDay,
	})

	// client.Business.PhoneNumber.FetchAll(manager.FetchPhoneNumberFilters{
	// 	GetSandboxNumbers: false,
	// })

	// client.Business.PhoneNumber.Fetch("113269274970227")
	// response, err := client.Business.Template.FetchAll()

	// response, err := client.Business.Template.Fetch(phoneNumber)

	textMessage, err := waboComponents.NewTextMessage(waboComponents.TextMessageConfigs{
		Text: "Hello, how can I help you?",
	})

	if err != nil {
		fmt.Println("error creating text message message", err)
		return
	}

	audioMessage, err := waboComponents.NewAudioMessage(waboComponents.AudioMessageConfigs{
		Link: "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3",
	})

	if err != nil {
		fmt.Println("error creating audio message message", err)
		return
	}

	videoMessage, err := waboComponents.NewVideoMessage(waboComponents.VideoMessageConfigs{
		Link: "https://www.sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4",
	})

	if err != nil {
		fmt.Println("error creating video message", err)
		return
	}

	imageMessage, err := waboComponents.NewImageMessage(waboComponents.ImageMessageConfigs{
		Link: "https://www.w3schools.com/w3css/img_lights.jpg",
	})

	if err != nil {
		fmt.Println("error creating image message", err)
		return
	}

	contactMessage, err := waboComponents.NewContactMessage([]waboComponents.Contact{
		*waboComponents.NewContact(waboComponents.ContactName{
			FormattedName: "Sarthak Jain",
			FirstName:     "Sarthak",
		})})

	if err != nil {
		fmt.Println("error creating contact message", err)
		return
	}

	locationMessage, err := waboComponents.NewLocationMessage(28.7041, 77.1025)

	if err != nil {
		fmt.Println("error creating location message", err)
		return
	}

	listMessage, err := waboComponents.NewListMessage(waboComponents.ListMessageParams{
		ButtonText: "Button 1",
		BodyText:   "Body 1",
	})

	if err != nil {
		fmt.Println("error creating list message", err)
		return
	}

	listSectionRow, err := waboComponents.NewListSectionRow("1", "Title 1", "Description 1")

	if err != nil {
		fmt.Println("error creating list section row", err)
		return
	}

	listSection, err := waboComponents.NewListSection("Section1")

	if err != nil {
		fmt.Println("error creating list section row", err)
		return
	}

	listSection.AddRow(listSectionRow)
	listMessage.AddSection(listSection)
	jsonData, err := listMessage.ToJson(waboComponents.ApiCompatibleJsonConverterConfigs{SendToPhoneNumber: "919643500545"})

	if err != nil {
		fmt.Println("error converting message to json", err)
		return
	}

	fmt.Println(string(jsonData))
	buttonMessage, err := waboComponents.NewQuickReplyButtonMessage("Body 1")

	if err != nil {
		fmt.Println("error creating button message", err)
		return
	}

	buttonMessage.AddButton("1", "Button 1")
	buttonMessage.AddButton("2", "Button 2")

	client.On(events.ReadyEventType, func(event events.BaseEvent) {
		fmt.Println("client is ready")
	})

	client.On(events.TextMessageEventType, func(event events.BaseEvent) {
		fmt.Println("text message event received")

		textMessageEvent := event.(*events.TextMessageEvent)

		switch strings.ToLower(textMessageEvent.Text) {
		case "text":
			textMessageEvent.Reply(textMessage)
		case "audio":
			textMessageEvent.Reply(audioMessage)
		case "image":
			textMessageEvent.Reply(imageMessage)
		case "video":
			textMessageEvent.Reply(videoMessage)
		case "contact":
			textMessageEvent.Reply(contactMessage)
		case "location":
			textMessageEvent.Reply(locationMessage)
		case "reaction":
			textMessageEvent.React("😍")
		case "list":
			textMessageEvent.Reply(listMessage)
		case "button":
			textMessageEvent.Reply(buttonMessage)
		case "qr":
			{
				response, err := client.Business.PhoneNumber.GenerateQrCode(phoneNumber, "This is Wabo this side.")

				if err != nil {
					fmt.Println("error generating qr code", err)
					return
				}

				fmt.Println("qr code response", response.QrImageUrl)

				qrCodeMessage, err := waboComponents.NewImageMessage(waboComponents.ImageMessageConfigs{
					Link: response.QrImageUrl,
					// Caption: "Scan the QR code to start a conversation.",
				})

				if err != nil {
					fmt.Println("error creating image message", err)
					return
				}

				textMessageEvent.Reply(qrCodeMessage)
			}
		default:
			textMessageEvent.Reply(textMessage)
		}
	})

	client.On(events.AudioMessageEventType, func(be events.BaseEvent) {
		fmt.Println("audio message event received")
	})

	client.On(events.VideoMessageEventType, func(be events.BaseEvent) {
		fmt.Println("video message event received")
	})

	client.On(events.DocumentMessageEventType, func(be events.BaseEvent) {
		fmt.Println("document message event received")
	})

	client.On(events.ImageMessageEventType, func(be events.BaseEvent) {
		fmt.Println("image message event received")
	})

	client.Initiate()
}
