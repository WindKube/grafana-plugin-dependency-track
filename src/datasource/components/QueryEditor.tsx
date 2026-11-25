import React from 'react';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { EditorField, EditorRow, EditorRows } from '@grafana/plugin-ui';
import { Combobox, ComboboxOption, InlineSwitch, Input, MultiCombobox, RadioButtonGroup } from '@grafana/ui';
import { DataSource } from '../datasource';
import { metricOptions } from '../metrics';
import {
  DEFAULT_QUERY,
  DTDataSourceOptions,
  DTQuery,
  ProjectSelector,
  QueryMode,
  QueryType,
  SelectorMode,
} from '../types';

type Props = QueryEditorProps<DataSource, DTQuery, DTDataSourceOptions>;

const queryTypes: Array<ComboboxOption<QueryType>> = [
  { value: QueryType.PortfolioMetrics, label: 'Portfolio metrics' },
  { value: QueryType.ProjectMetrics, label: 'Project metrics' },
];

const modes: Array<SelectableValue<QueryMode>> = [
  { value: QueryMode.Current, label: 'Current' },
  { value: QueryMode.TimeSeries, label: 'Time series' },
];

const selectorModes: Array<ComboboxOption<SelectorMode>> = [
  { value: 'all', label: 'All projects' },
  { value: 'name', label: 'Name regex' },
  { value: 'tag', label: 'Tag' },
  { value: 'classifier', label: 'Classifier' },
  { value: 'uuid', label: 'UUID' },
];

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const q = { ...DEFAULT_QUERY, ...query };
  const projects: ProjectSelector = { ...DEFAULT_QUERY.projects, ...query.projects };
  const isPortfolio = q.queryType === QueryType.PortfolioMetrics;

  const update = (patch: Partial<DTQuery>) => {
    onChange({ ...query, ...patch });
    onRunQuery();
  };
  const updateProjects = (patch: Partial<ProjectSelector>) => update({ projects: { ...projects, ...patch } });

  return (
    <EditorRows>
      <EditorRow>
        <EditorField label="Query type">
          <Combobox
            aria-label="Query type"
            width={24}
            options={queryTypes}
            value={q.queryType}
            onChange={(v) => {
              const metrics = (q.metrics ?? []).filter((m) =>
                metricOptions(v.value === QueryType.PortfolioMetrics).some((o) => o.value === m)
              );
              update({ queryType: v.value, metrics });
            }}
          />
        </EditorField>
        <EditorField label="Mode">
          <RadioButtonGroup options={modes} value={q.mode} onChange={(mode) => update({ mode })} />
        </EditorField>
        <EditorField label="Metrics" width={60}>
          <MultiCombobox
            aria-label="Metrics"
            options={metricOptions(isPortfolio)}
            value={q.metrics}
            onChange={(v) => update({ metrics: v.map((o) => o.value) })}
          />
        </EditorField>
      </EditorRow>
      {!isPortfolio && (
        <EditorRow>
          <EditorField label="Projects">
            <Combobox
              aria-label="Project selector"
              width={20}
              options={selectorModes}
              value={projects.mode}
              onChange={(v) => updateProjects({ mode: v.value })}
            />
          </EditorField>
          {projects.mode === 'name' && (
            <EditorField label="Name regex">
              <Input
                aria-label="Name regex"
                width={30}
                defaultValue={projects.nameRegex}
                placeholder="^payments-.*"
                onBlur={(e) => updateProjects({ nameRegex: e.currentTarget.value })}
              />
            </EditorField>
          )}
          {projects.mode === 'tag' && (
            <EditorField label="Tag">
              <Input
                aria-label="Tag"
                width={30}
                defaultValue={projects.tag}
                onBlur={(e) => updateProjects({ tag: e.currentTarget.value })}
              />
            </EditorField>
          )}
          {projects.mode === 'classifier' && (
            <EditorField label="Classifier">
              <Input
                aria-label="Classifier"
                width={30}
                defaultValue={projects.classifier}
                placeholder="APPLICATION"
                onBlur={(e) => updateProjects({ classifier: e.currentTarget.value })}
              />
            </EditorField>
          )}
          {projects.mode === 'uuid' && (
            <EditorField label="UUIDs">
              <Input
                aria-label="UUIDs"
                width={60}
                defaultValue={projects.uuids?.join(',')}
                placeholder="comma-separated project UUIDs"
                onBlur={(e) =>
                  updateProjects({
                    uuids: e.currentTarget.value
                      .split(',')
                      .map((s) => s.trim())
                      .filter(Boolean),
                  })
                }
              />
            </EditorField>
          )}
          <EditorField label="Latest version only">
            <InlineSwitch
              aria-label="Latest version only"
              value={projects.onlyLatest}
              onChange={(e) => updateProjects({ onlyLatest: e.currentTarget.checked })}
            />
          </EditorField>
          <EditorField label="Include inactive">
            <InlineSwitch
              aria-label="Include inactive"
              value={projects.includeInactive}
              onChange={(e) => updateProjects({ includeInactive: e.currentTarget.checked })}
            />
          </EditorField>
        </EditorRow>
      )}
    </EditorRows>
  );
}
