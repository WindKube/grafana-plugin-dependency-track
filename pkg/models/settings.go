package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

const (
	DefaultCacheTTL      = 60 * time.Second
	DefaultMaxConcurrent = 10
)

type Settings struct {
	URL    string `json:"-"`
	UIURL  string `json:"uiUrl"`
	APIKey string `json:"-"`
	// CustomTimeout is true when jsonData sets Grafana's HTTP timeout; otherwise the plugin
	// default applies instead of the SDK's 30s.
	CustomTimeout bool `json:"-"`
	// CacheTTLSeconds is nil when unset; 0 disables the cache.
	CacheTTLSeconds *int `json:"cacheTtl,omitempty"`
	MaxConcurrent   int  `json:"maxConcurrentRequests,omitempty"`
}

func (s *Settings) CacheTTL() time.Duration {
	if s.CacheTTLSeconds == nil {
		return DefaultCacheTTL
	}
	return time.Duration(max(*s.CacheTTLSeconds, 0)) * time.Second
}

func (s *Settings) Concurrency() int {
	if s.MaxConcurrent <= 0 {
		return DefaultMaxConcurrent
	}
	return s.MaxConcurrent
}

func LoadSettings(source backend.DataSourceInstanceSettings) (*Settings, error) {
	s := Settings{}
	if len(source.JSONData) > 0 {
		if err := json.Unmarshal(source.JSONData, &s); err != nil {
			return nil, fmt.Errorf("could not parse settings: %w", err)
		}
		// Same rule as the SDK: only a positive number overrides the default.
		var raw struct {
			Timeout any `json:"timeout"`
		}
		_ = json.Unmarshal(source.JSONData, &raw)
		if v, ok := raw.Timeout.(float64); ok && v > 0 {
			s.CustomTimeout = true
		}
	}
	s.URL = strings.TrimSpace(source.URL)
	s.UIURL = strings.TrimRight(strings.TrimSpace(s.UIURL), "/")
	s.APIKey = source.DecryptedSecureJSONData["apiKey"]
	return &s, nil
}

func (s *Settings) Validate() error {
	if s.URL == "" {
		return errors.New("Dependency-Track URL is not configured")
	}
	if s.APIKey == "" {
		return errors.New("API key is not configured")
	}
	return nil
}
