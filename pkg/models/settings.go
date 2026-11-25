package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Settings struct {
	URL    string `json:"-"`
	UIURL  string `json:"uiUrl"`
	APIKey string `json:"-"`
}

func LoadSettings(source backend.DataSourceInstanceSettings) (*Settings, error) {
	s := Settings{}
	if len(source.JSONData) > 0 {
		if err := json.Unmarshal(source.JSONData, &s); err != nil {
			return nil, fmt.Errorf("could not parse settings: %w", err)
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
