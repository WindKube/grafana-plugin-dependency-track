import { ComboboxOption } from '@grafana/ui';

interface MetricDef {
  value: string;
  label: string;
  portfolioOnly?: boolean;
}

export const METRICS: MetricDef[] = [
  { value: 'critical', label: 'Critical' },
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
  { value: 'unassigned', label: 'Unassigned' },
  { value: 'vulnerabilities', label: 'Vulnerabilities' },
  { value: 'kev', label: 'Known exploited (KEV)' },
  { value: 'components', label: 'Components' },
  { value: 'vulnerableComponents', label: 'Vulnerable components' },
  { value: 'suppressed', label: 'Suppressed' },
  { value: 'findingsTotal', label: 'Findings' },
  { value: 'findingsAudited', label: 'Findings audited' },
  { value: 'findingsUnaudited', label: 'Findings unaudited' },
  { value: 'inheritedRiskScore', label: 'Inherited risk score' },
  { value: 'policyViolationsFail', label: 'Policy violations (fail)' },
  { value: 'policyViolationsWarn', label: 'Policy violations (warn)' },
  { value: 'policyViolationsInfo', label: 'Policy violations (info)' },
  { value: 'policyViolationsTotal', label: 'Policy violations' },
  { value: 'policyViolationsUnaudited', label: 'Policy violations unaudited' },
  { value: 'policyViolationsSecurityTotal', label: 'Security violations' },
  { value: 'policyViolationsLicenseTotal', label: 'License violations' },
  { value: 'policyViolationsOperationalTotal', label: 'Operational violations' },
  { value: 'projects', label: 'Projects', portfolioOnly: true },
  { value: 'vulnerableProjects', label: 'Vulnerable projects', portfolioOnly: true },
];

export function metricOptions(portfolio: boolean): Array<ComboboxOption<string>> {
  return METRICS.filter((m) => portfolio || !m.portfolioOnly).map(({ value, label }) => ({ value, label }));
}
