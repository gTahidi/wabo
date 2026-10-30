package manager

import (
	"encoding/json"
	"testing"

	"github.com/gTahidi/wabo/pkg/events"
)

func TestWebhookPreservesMessageContent(t *testing.T) {
	cases := []struct {
		name, message string
		kind          events.EventType
		check         func(*testing.T, events.BaseEvent)
	}{
		{"referral", `{"type":"text","text":{"body":"hello"},"referral":{"source_id":"ad-1","future":true}}`, events.TextMessageEventType, func(t *testing.T, ev events.BaseEvent) {
			e := ev.(*events.TextMessageEvent)
			if len(e.Referral) == 0 || len(e.RawMessage) == 0 {
				t.Fatal("lost attribution or raw message")
			}
			var referral map[string]interface{}
			if err := json.Unmarshal(e.Referral, &referral); err != nil || referral["source_id"] != "ad-1" || referral["future"] != true {
				t.Fatalf("referral %s", e.Referral)
			}
		}},
		{"document", `{"type":"document","document":{"id":"doc-1","mime_type":"application/pdf","sha256":"hash","filename":"report.pdf","caption":"report"}}`, events.DocumentMessageEventType, func(t *testing.T, ev events.BaseEvent) {
			e, ok := ev.(*events.DocumentMessageEvent)
			if !ok {
				t.Fatalf("got %T", ev)
			}
			if e.MediaId != "doc-1" || e.MimeType != "application/pdf" || e.Sha256 != "hash" || e.Document.FileName != "report.pdf" || e.Document.Caption == nil || *e.Document.Caption != "report" {
				t.Fatalf("document %+v", e)
			}
		}},
		{"document without filename", `{"type":"document","document":{"id":"doc-2"}}`, events.DocumentMessageEventType, func(t *testing.T, ev events.BaseEvent) {
			if ev.(*events.DocumentMessageEvent).MediaId != "doc-2" {
				t.Fatal("lost document")
			}
		}},
		{"contact cards", `{"type":"contacts","contacts":[{"name":{"formatted_name":"Alice"},"phones":[{"phone":"123","wa_id":"123","type":"CELL"}]}]}`, events.ContactMessageEventType, func(t *testing.T, ev events.BaseEvent) {
			e := ev.(*events.ContactsMessageEvent)
			if len(e.Contacts.Contacts) != 1 || e.Contacts.Contacts[0].Name.FormattedName != "Alice" || e.Contacts.Contacts[0].Phones[0].WaId != "123" {
				t.Fatalf("contacts %+v", e)
			}
		}},
		{"unknown system", `{"type":"system","system":{"type":"user_changed_user_id","user_id":"new"}}`, events.UnknownEventType, func(t *testing.T, ev events.BaseEvent) {
			if len(ev.(*events.UnknownWebhookEvent).Raw) == 0 {
				t.Fatal("lost system payload")
			}
		}},
		{"unknown message", `{"type":"future_type","future":{"value":1}}`, events.UnknownEventType, func(t *testing.T, ev events.BaseEvent) {
			if len(ev.(*events.UnknownWebhookEvent).Raw) == 0 {
				t.Fatal("lost future payload")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wh := newTestWebhook(t, "")
			ch, _ := wh.EventManager.Subscribe(tc.kind)
			body := `{"entry":[{"id":"business","changes":[{"field":"messages","value":{"messages":[` + tc.message + `]}}]}]}`
			if rec := post(wh, body, ""); rec.Code != 200 {
				t.Fatalf("status %d: %s", rec.Code, rec.Body)
			}
			select {
			case ev := <-ch:
				tc.check(t, ev.Data)
			default:
				t.Fatal("no event")
			}
		})
	}
}

func TestBatchSenderNames(t *testing.T) {
	wh := newTestWebhook(t, "")
	ch, _ := wh.EventManager.Subscribe(events.TextMessageEventType)
	body := `{"entry":[{"changes":[{"field":"messages","value":{"contacts":[{"wa_id":"2","profile":{"name":"Bob"}},{"wa_id":"1","profile":{"name":"Alice"}}],"messages":[{"from":"1","type":"text","text":{"body":"a"}},{"from":"2","type":"text","text":{"body":"b"}},{"from":"3","type":"text","text":{"body":"c"}}]}}]}]}`
	if rec := post(wh, body, ""); rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	for _, name := range []string{"Alice", "Bob", ""} {
		select {
		case ev := <-ch:
			if got := ev.Data.(*events.TextMessageEvent).SenderName; got != name {
				t.Fatalf("sender %q want %q", got, name)
			}
		default:
			t.Fatal("missing message")
		}
	}
}

func TestUnknownChangePreservesLargeID(t *testing.T) {
	wh := newTestWebhook(t, "")
	ch, _ := wh.EventManager.Subscribe(events.UnknownEventType)
	body := `{"entry":[{"id":"b","changes":[{"field":"future","value":{"id":9007199254740993}}]}]}`
	post(wh, body, "")
	select {
	case ev := <-ch:
		e := ev.Data.(*events.UnknownWebhookEvent)
		if string(e.Raw) != `{"id":9007199254740993}` || e.Field != "future" || e.BusinessAccountId != "b" {
			t.Fatalf("unknown %+v", e)
		}
	default:
		t.Fatal("no unknown event")
	}
}

func TestPricingBillablePresence(t *testing.T) {
	for _, tc := range []struct {
		body    string
		present bool
	}{{`{"billable":false,"type":"regular"}`, true}, {`{}`, false}} {
		var p Pricing
		if err := json.Unmarshal([]byte(tc.body), &p); err != nil {
			t.Fatal(err)
		}
		if (p.Billable != nil) != tc.present || (p.Billable != nil && *p.Billable) {
			t.Fatalf("pricing %+v", p)
		}
	}
}
