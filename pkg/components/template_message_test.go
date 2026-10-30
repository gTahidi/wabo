package components

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewAuthenticationTemplateMessageValidation(t *testing.T) {
	for _, tc := range []struct{ name, lang, code string }{
		{"", "en", "1"}, {"code", "", "1"}, {"code", "en", ""}, {"code", "en", strings.Repeat("9", 16)},
	} {
		if _, err := NewAuthenticationTemplateMessage(tc.name, tc.lang, tc.code); err == nil {
			t.Fatalf("%+v: want error", tc)
		}
	}
}

func TestAddBodyTextOverridesBody(t *testing.T) {
	tm, _ := NewTemplateMessage(&TemplateMessageConfigs{Name: "dd_quote_ready", Language: "en"})
	tm.AddBodyText("old")
	tm.AddBodyText("Amina", "KES 12,000")
	b, err := tm.ToJson(ApiCompatibleJsonConverterConfigs{SendToPhoneNumber: "+254700000000"})
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Template struct {
			Components []struct {
				Type       string
				Parameters []struct{ Type, Text string }
			}
		}
	}
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	c := p.Template.Components
	if len(c) != 1 || c[0].Type != "body" || len(c[0].Parameters) != 2 || c[0].Parameters[1].Text != "KES 12,000" {
		t.Fatalf("unexpected payload %s", b)
	}
}

func TestTemplateButtonIndexRange(t *testing.T) {
	for _, index := range []int{-1, 0, 9, 10} {
		tm, err := NewTemplateMessage(&TemplateMessageConfigs{Name: "test", Language: "en"})
		if err != nil {
			t.Fatal(err)
		}
		code := "123"
		err = tm.AddButton(TemplateMessageComponentButtonType{Type: TemplateMessageComponentTypeButton, SubType: TemplateMessageButtonComponentTypeUrl, Index: index, Parameters: &[]TemplateMessageParameter{TemplateMessageButtonParameter{Type: TemplateMessageButtonParameterTypeText, Text: &code}}})
		if wantValid := index >= 0 && index <= 9; (err == nil) != wantValid {
			t.Fatalf("index %d error %v", index, err)
		}
	}
}
