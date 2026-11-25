import React from 'react';
import { QueryEditorProps } from '@grafana/data';
import { EditorField, EditorRow, EditorRows } from '@grafana/plugin-ui';
import { Combobox, ComboboxOption, InlineSwitch, Input } from '@grafana/ui';

import type { DataSource } from '../datasource';
import { DTDataSourceOptions, DTQuery, DTVariableQuery, VariableQueryType } from '../types';

type Props = QueryEditorProps<DataSource, DTQuery, DTDataSourceOptions, DTVariableQuery>;

const types: Array<ComboboxOption<VariableQueryType>> = [
  { value: 'projects', label: 'Project names' },
  { value: 'versions', label: 'Versions of a project' },
  { value: 'tags', label: 'Tags' },
  { value: 'classifiers', label: 'Classifiers' },
  { value: 'policies', label: 'Policies' },
];

export function VariableQueryEditor({ query, onChange }: Props) {
  const q: DTVariableQuery = { ...query, refId: query.refId ?? 'A', type: query.type ?? 'projects' };
  return (
    <EditorRows>
      <EditorRow>
        <EditorField label="Values">
          <Combobox
            aria-label="Variable values"
            width={30}
            options={types}
            value={q.type}
            onChange={(v) => onChange({ ...q, type: v.value })}
          />
        </EditorField>
        {q.type === 'versions' && (
          <EditorField label="Project" tooltip="Project name; template variables such as $project are supported.">
            <Input
              aria-label="Project"
              width={30}
              defaultValue={q.project}
              placeholder="$project"
              onBlur={(e) => onChange({ ...q, project: e.currentTarget.value })}
            />
          </EditorField>
        )}
        {q.type === 'projects' && (
          <EditorField label="Latest version only">
            <InlineSwitch
              aria-label="Latest version only"
              value={q.onlyLatest}
              onChange={(e) => onChange({ ...q, onlyLatest: e.currentTarget.checked })}
            />
          </EditorField>
        )}
      </EditorRow>
    </EditorRows>
  );
}
