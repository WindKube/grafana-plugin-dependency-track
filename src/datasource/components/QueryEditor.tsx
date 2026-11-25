import React, { useEffect, useState } from 'react';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { EditorField, EditorRow, EditorRows } from '@grafana/plugin-ui';
import { getTemplateSrv } from '@grafana/runtime';
import { Combobox, ComboboxOption, InlineSwitch, Input, MultiCombobox, RadioButtonGroup } from '@grafana/ui';
import { DataSource } from '../datasource';
import { metricOptions } from '../metrics';
import {
  DEFAULT_QUERY,
  DTDataSourceOptions,
  DTQuery,
  ProjectSelector,
  QueryFormat,
  QueryMode,
  QueryType,
  SelectorMode,
} from '../types';
import { FindingFilterEditor, ViolationFilterEditor } from './FilterEditors';

type Props = QueryEditorProps<DataSource, DTQuery, DTDataSourceOptions>;
type Options = Array<ComboboxOption<string>>;

const queryTypes: Array<ComboboxOption<QueryType>> = [
  { value: QueryType.PortfolioMetrics, label: 'Portfolio metrics' },
  { value: QueryType.ProjectMetrics, label: 'Project metrics' },
  { value: QueryType.Findings, label: 'Findings' },
  { value: QueryType.Violations, label: 'Policy violations' },
];

const formats: Array<SelectableValue<QueryFormat>> = [
  { value: 'table', label: 'Table', description: 'One row per finding or violation' },
  { value: 'count', label: 'Count', description: 'Counts per project for alerting' },
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
  { value: 'uuid', label: 'Pick projects' },
];

function useOptions(load: () => Promise<Options>, enabled: boolean): Options {
  const [options, setOptions] = useState<Options>([]);
  useEffect(() => {
    if (enabled) {
      load()
        .then(setOptions)
        .catch(() => setOptions([]));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled]);
  return options;
}

function withVariables(options: Options): Options {
  const variables = getTemplateSrv()
    .getVariables()
    .map((v) => '$' + v.name);
  return [...variables.map((name) => ({ value: name, group: 'Variables' })), ...options];
}

export function QueryEditor({ datasource, query, onChange, onRunQuery }: Props) {
  const q = { ...DEFAULT_QUERY, ...query };
  const projects: ProjectSelector = { ...DEFAULT_QUERY.projects, ...query.projects };
  const isPortfolio = q.queryType === QueryType.PortfolioMetrics;
  const isMetrics = isPortfolio || q.queryType === QueryType.ProjectMetrics;
  const isViolations = q.queryType === QueryType.Violations;
  const format = q.format ?? 'table';
  const policyOptions = useOptions(
    async () => (await datasource.getPolicies()).map((value) => ({ value })),
    isViolations
  );

  const tagOptions = useOptions(
    async () => (await datasource.getTags()).map((value) => ({ value })),
    !isPortfolio && projects.mode === 'tag'
  );
  const classifierOptions = useOptions(
    async () => (await datasource.getClassifiers()).map((value) => ({ value })),
    !isPortfolio && projects.mode === 'classifier'
  );
  const projectOptions = useOptions(
    async () =>
      (await datasource.getProjects()).map((p) => ({
        value: p.uuid,
        label: `${p.name} ${p.version}`,
        description: p.isLatest ? 'latest' : undefined,
      })),
    !isPortfolio && projects.mode === 'uuid'
  );

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
        {isMetrics ? (
          <>
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
          </>
        ) : (
          <EditorField label="Format">
            <RadioButtonGroup options={formats} value={format} onChange={(f) => update({ format: f })} />
          </EditorField>
        )}
      </EditorRow>
      {q.queryType === QueryType.Findings && (
        <FindingFilterEditor filter={q.findings ?? {}} onChange={(findings) => update({ findings })} />
      )}
      {isViolations && (
        <ViolationFilterEditor
          filter={q.violations ?? {}}
          policies={withVariables(policyOptions)}
          showGrouping={format === 'count'}
          onChange={(violations) => update({ violations })}
        />
      )}
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
            <EditorField label="Name regex" tooltip="Multi-value variables are expanded to an alternation.">
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
              <Combobox
                aria-label="Tag"
                width={30}
                createCustomValue
                options={withVariables(tagOptions)}
                value={projects.tag ?? null}
                onChange={(v) => updateProjects({ tag: v.value })}
              />
            </EditorField>
          )}
          {projects.mode === 'classifier' && (
            <EditorField label="Classifier">
              <Combobox
                aria-label="Classifier"
                width={30}
                createCustomValue
                options={withVariables(classifierOptions)}
                value={projects.classifier ?? null}
                onChange={(v) => updateProjects({ classifier: v.value })}
              />
            </EditorField>
          )}
          {projects.mode === 'uuid' && (
            <EditorField label="Projects" width={60}>
              <MultiCombobox
                aria-label="Projects"
                createCustomValue
                options={withVariables(projectOptions)}
                value={projects.uuids ?? []}
                onChange={(v) => updateProjects({ uuids: v.map((o) => o.value) })}
              />
            </EditorField>
          )}
          {projects.mode !== 'uuid' && (
            <>
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
            </>
          )}
        </EditorRow>
      )}
    </EditorRows>
  );
}
