package plugin

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
	"golang.org/x/sync/errgroup"
)

const maxConcurrency = 8

func (d *Datasource) resolveProjects(ctx context.Context, sel query.ProjectSelector) ([]dtrack.Project, error) {
	filter := dtrack.ProjectFilter{ExcludeInactive: !sel.IncludeInactive}
	var projects []dtrack.Project

	switch sel.Mode {
	case query.SelectUUID:
		projects = make([]dtrack.Project, len(sel.UUIDs))
		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(maxConcurrency)
		for i, uuid := range sel.UUIDs {
			g.Go(func() error {
				p, err := d.client.Project(gctx, uuid)
				if err != nil {
					return err
				}
				projects[i] = *p
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
	case query.SelectTag:
		filter.Tag = sel.Tag
	case query.SelectClassifier:
		filter.Classifier = sel.Classifier
	}

	if projects == nil {
		var err error
		if projects, err = d.client.Projects(ctx, filter); err != nil {
			return nil, err
		}
	}

	var nameRe *regexp.Regexp
	if sel.Mode == query.SelectName && sel.NameRegex != "" {
		nameRe = regexp.MustCompile(sel.NameRegex)
	}
	// Explicitly picked projects are returned as-is, even when they are not the latest version
	// or are inactive.
	explicit := sel.Mode == query.SelectUUID
	out := projects[:0]
	for _, p := range projects {
		if nameRe != nil && !nameRe.MatchString(p.Name) {
			continue
		}
		if !explicit && sel.OnlyLatest && !p.IsLatest {
			continue
		}
		if !explicit && !sel.IncludeInactive && !p.Active {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Version < out[j].Version
	})
	return out, nil
}

func projectLabels(p dtrack.Project) map[string]string {
	tags := p.TagNames()
	sort.Strings(tags)
	return map[string]string{
		"project": p.Name,
		"version": p.Version,
		"tags":    strings.Join(tags, ","),
	}
}
