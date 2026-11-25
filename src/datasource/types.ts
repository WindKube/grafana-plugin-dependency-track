import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export enum QueryType {
  PortfolioMetrics = 'portfolioMetrics',
  ProjectMetrics = 'projectMetrics',
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
