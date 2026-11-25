import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export enum QueryType {
  PortfolioMetrics = 'portfolioMetrics',
  ProjectMetrics = 'projectMetrics',
  Findings = 'findings',
  Violations = 'violations',
  Projects = 'projects',
  VulnerabilityMetrics = 'vulnerabilityMetrics',
  Events = 'events',
}

export const EVENT_KINDS = [
  { value: 'bomImport', label: 'BOM imports' },
  { value: 'finding', label: 'New findings' },
  { value: 'violation', label: 'Policy violations' },
];

export interface EventFilter {
  kinds?: string[];
}

export type QueryFormat = 'table' | 'count' | 'bomAge';

export const SEVERITIES = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'UNASSIGNED'];
export const ANALYSIS_STATES = ['NOT_SET', 'EXPLOITABLE', 'IN_TRIAGE', 'FALSE_POSITIVE', 'NOT_AFFECTED', 'RESOLVED'];
export const VIOLATION_STATES = ['FAIL', 'WARN', 'INFO'];
export const RISK_TYPES = ['SECURITY', 'LICENSE', 'OPERATIONAL'];

export interface FindingFilter {
  severities?: string[];
  analysisStates?: string[];
  includeSuppressed?: boolean;
  cvssMin?: number;
  epssMin?: number;
  kevOnly?: boolean;
}

export interface ViolationFilter {
  states?: string[];
  riskTypes?: string[];
  policies?: string[];
  includeSuppressed?: boolean;
  groupByPolicy?: boolean;
}

export enum QueryMode {
  Current = 'current',
  TimeSeries = 'timeseries',
}

export type SelectorMode = 'all' | 'uuid' | 'name' | 'tag' | 'classifier';

export interface ProjectSelector {
  mode?: SelectorMode;
  uuids?: string[];
  nameRegex?: string;
  tag?: string;
  classifier?: string;
  onlyLatest?: boolean;
  includeInactive?: boolean;
}

export interface DTQuery extends DataQuery {
  queryType?: QueryType;
  mode?: QueryMode;
  metrics?: string[];
  projects?: ProjectSelector;
  format?: QueryFormat;
  findings?: FindingFilter;
  violations?: ViolationFilter;
  events?: EventFilter;
}

export const DEFAULT_METRICS = ['critical', 'high', 'medium', 'low', 'unassigned'];

export const DEFAULT_QUERY: Partial<DTQuery> = {
  queryType: QueryType.PortfolioMetrics,
  mode: QueryMode.Current,
  metrics: DEFAULT_METRICS,
  projects: { mode: 'all', onlyLatest: true },
};

export interface DTDataSourceOptions extends DataSourceJsonData {
  uiUrl?: string;
  timeout?: number;
  enableSecureSocksProxy?: boolean;
}

export interface DTSecureJsonData {
  apiKey?: string;
}

export type VariableQueryType = 'projects' | 'versions' | 'tags' | 'classifiers' | 'policies';

export interface DTVariableQuery extends DataQuery {
  type: VariableQueryType;
  project?: string;
  onlyLatest?: boolean;
}

export interface ProjectOption {
  uuid: string;
  name: string;
  version: string;
  isLatest: boolean;
  tags: string[];
}
