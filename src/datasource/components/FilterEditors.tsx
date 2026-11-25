import React from 'react';
import { EditorField, EditorRow } from '@grafana/plugin-ui';
import { ComboboxOption, InlineSwitch, Input, MultiCombobox } from '@grafana/ui';

import { ANALYSIS_STATES, FindingFilter, RISK_TYPES, SEVERITIES, VIOLATION_STATES, ViolationFilter } from '../types';

const toOptions = (values: string[]): Array<ComboboxOption<string>> => values.map((value) => ({ value }));

const parseNumber = (value: string) => (value.trim() === '' ? undefined : Number(value));

interface FindingProps {
  filter: FindingFilter;
  onChange: (filter: FindingFilter) => void;
}

export function FindingFilterEditor({ filter, onChange }: FindingProps) {
  const update = (patch: Partial<FindingFilter>) => onChange({ ...filter, ...patch });
  return (
    <EditorRow>
      <EditorField label="Severity" tooltip="All severities when empty">
        <MultiCombobox
          aria-label="Severity"
          width={40}
          options={toOptions(SEVERITIES)}
          value={filter.severities ?? []}
          onChange={(v) => update({ severities: v.map((o) => o.value) })}
        />
      </EditorField>
      <EditorField label="Analysis state" tooltip="All states when empty">
        <MultiCombobox
          aria-label="Analysis state"
          width={40}
          options={toOptions(ANALYSIS_STATES)}
          value={filter.analysisStates ?? []}
          onChange={(v) => update({ analysisStates: v.map((o) => o.value) })}
        />
      </EditorField>
      <EditorField label="Min CVSS">
        <Input
          aria-label="Min CVSS"
          type="number"
          width={10}
          min={0}
          max={10}
          step={0.1}
          defaultValue={filter.cvssMin}
          onBlur={(e) => update({ cvssMin: parseNumber(e.currentTarget.value) })}
        />
      </EditorField>
      <EditorField label="Min EPSS">
        <Input
          aria-label="Min EPSS"
          type="number"
          width={10}
          min={0}
          max={1}
          step={0.01}
          defaultValue={filter.epssMin}
          onBlur={(e) => update({ epssMin: parseNumber(e.currentTarget.value) })}
        />
      </EditorField>
      <EditorField label="KEV only">
        <InlineSwitch
          aria-label="KEV only"
          value={filter.kevOnly}
          onChange={(e) => update({ kevOnly: e.currentTarget.checked })}
        />
      </EditorField>
      <EditorField label="Include suppressed">
        <InlineSwitch
          aria-label="Include suppressed"
          value={filter.includeSuppressed}
          onChange={(e) => update({ includeSuppressed: e.currentTarget.checked })}
        />
      </EditorField>
    </EditorRow>
  );
}

interface ViolationProps {
  filter: ViolationFilter;
  policies: Array<ComboboxOption<string>>;
  showGrouping: boolean;
  onChange: (filter: ViolationFilter) => void;
}

export function ViolationFilterEditor({ filter, policies, showGrouping, onChange }: ViolationProps) {
  const update = (patch: Partial<ViolationFilter>) => onChange({ ...filter, ...patch });
  return (
    <EditorRow>
      <EditorField label="State" tooltip="All states when empty">
        <MultiCombobox
          aria-label="Violation state"
          width={30}
          options={toOptions(VIOLATION_STATES)}
          value={filter.states ?? []}
          onChange={(v) => update({ states: v.map((o) => o.value) })}
        />
      </EditorField>
      <EditorField label="Risk type" tooltip="All risk types when empty">
        <MultiCombobox
          aria-label="Risk type"
          width={36}
          options={toOptions(RISK_TYPES)}
          value={filter.riskTypes ?? []}
          onChange={(v) => update({ riskTypes: v.map((o) => o.value) })}
        />
      </EditorField>
      <EditorField label="Policy" tooltip="All policies when empty">
        <MultiCombobox
          aria-label="Policy"
          width={40}
          createCustomValue
          options={policies}
          value={filter.policies ?? []}
          onChange={(v) => update({ policies: v.map((o) => o.value) })}
        />
      </EditorField>
      {showGrouping && (
        <EditorField label="Group by policy">
          <InlineSwitch
            aria-label="Group by policy"
            value={filter.groupByPolicy}
            onChange={(e) => update({ groupByPolicy: e.currentTarget.checked })}
          />
        </EditorField>
      )}
      <EditorField label="Include suppressed">
        <InlineSwitch
          aria-label="Include suppressed"
          value={filter.includeSuppressed}
          onChange={(e) => update({ includeSuppressed: e.currentTarget.checked })}
        />
      </EditorField>
    </EditorRow>
  );
}
