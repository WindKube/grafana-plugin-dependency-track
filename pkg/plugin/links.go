package plugin

import (
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type fieldLink struct {
	title string
	path  string
	needs []string
}

// tableLinks maps table columns to Dependency-Track UI pages. A link is only added when every
// column it interpolates is present in the frame.
var tableLinks = map[string][]fieldLink{
	"Project": {{
		title: "Open project in Dependency-Track",
		path:  `/projects/${__data.fields["Project UUID"]}`,
		needs: []string{"Project UUID"},
	}},
	"Component": {{
		title: "Open component in Dependency-Track",
		path:  `/components/${__data.fields["Component UUID"]}`,
		needs: []string{"Component UUID"},
	}},
	"Vulnerability": {
		{
			title: "Open finding in Dependency-Track",
			path:  `/projects/${__data.fields["Project UUID"]}/findings/${__data.fields["Component UUID"]}/${__data.fields["Vulnerability UUID"]}`,
			needs: []string{"Project UUID", "Component UUID", "Vulnerability UUID"},
		},
		{
			title: "Open vulnerability in Dependency-Track",
			path:  `/vulnerabilities/${__data.fields["Source"]}/${__value.raw}`,
			needs: []string{"Source"},
		},
	},
	"Policy": {{
		title: "Open project policy violations in Dependency-Track",
		path:  `/projects/${__data.fields["Project UUID"]}/policyViolations`,
		needs: []string{"Project UUID"},
	}},
}

func addLinks(f *data.Frame, uiURL string) {
	uiURL = strings.TrimRight(uiURL, "/")
	if uiURL == "" {
		return
	}
	names := map[string]bool{}
	for _, field := range f.Fields {
		names[field.Name] = true
	}
	for _, field := range f.Fields {
		for _, l := range tableLinks[field.Name] {
			if !hasAll(names, l.needs) {
				continue
			}
			if field.Config == nil {
				field.Config = &data.FieldConfig{}
			}
			field.Config.Links = append(field.Config.Links, data.DataLink{
				Title:       l.title,
				URL:         uiURL + l.path,
				TargetBlank: true,
			})
		}
	}
}

func hasAll(names map[string]bool, keys []string) bool {
	for _, k := range keys {
		if !names[k] {
			return false
		}
	}
	return true
}
