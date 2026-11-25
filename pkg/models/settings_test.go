package models

import (
	"testing"
	"time"

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

func TestCustomTimeout(t *testing.T) {
	for raw, want := range map[string]bool{
		`{}`:               false,
		`{"timeout":0}`:    false,
		`{"timeout":"30"}`: false,
		`{"timeout":120}`:  true,
	} {
		s, err := LoadSettings(backend.DataSourceInstanceSettings{JSONData: []byte(raw)})
		if err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if s.CustomTimeout != want {
			t.Errorf("%s: CustomTimeout = %v", raw, s.CustomTimeout)
		}
	}
}

func TestCacheAndConcurrencyDefaults(t *testing.T) {
	for _, tc := range []struct {
		json        string
		ttl         time.Duration
		concurrency int
	}{
		{`{}`, DefaultCacheTTL, DefaultMaxConcurrent},
		{`{"cacheTtl":0,"maxConcurrentRequests":3}`, 0, 3},
		{`{"cacheTtl":300}`, 5 * time.Minute, DefaultMaxConcurrent},
		{`{"cacheTtl":-5,"maxConcurrentRequests":-1}`, 0, DefaultMaxConcurrent},
	} {
		s, err := LoadSettings(backend.DataSourceInstanceSettings{JSONData: []byte(tc.json)})
		if err != nil {
			t.Fatal(err)
		}
		if s.CacheTTL() != tc.ttl || s.Concurrency() != tc.concurrency {
			t.Errorf("%s: ttl %v concurrency %d", tc.json, s.CacheTTL(), s.Concurrency())
		}
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
