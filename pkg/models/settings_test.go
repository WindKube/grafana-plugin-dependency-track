package models

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestLoadSettings(t *testing.T) {
	s, err := LoadSettings(backend.DataSourceInstanceSettings{
		URL:                     " https://dt.example.com ",
		JSONData:                []byte(`{"uiUrl":"https://dt-ui.example.com/"}`),
		DecryptedSecureJSONData: map[string]string{"apiKey": "odt_x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.URL != "https://dt.example.com" || s.UIURL != "https://dt-ui.example.com" || s.APIKey != "odt_x" {
		t.Fatalf("got %+v", s)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidate(t *testing.T) {
	if err := (&Settings{APIKey: "k"}).Validate(); err == nil {
		t.Error("expected missing URL error")
	}
	if err := (&Settings{URL: "http://dt"}).Validate(); err == nil {
		t.Error("expected missing API key error")
	}
}

func TestLoadSettingsInvalidJSON(t *testing.T) {
	if _, err := LoadSettings(backend.DataSourceInstanceSettings{JSONData: []byte(`{`)}); err == nil {
		t.Error("expected error")
	}
}
