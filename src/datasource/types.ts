import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface DTQuery extends DataQuery {
  queryText?: string;
  constant: number;
}

export const DEFAULT_QUERY: Partial<DTQuery> = {
  constant: 6.5,
};

export interface DTDataSourceOptions extends DataSourceJsonData {
  uiUrl?: string;
  timeout?: number;
  enableSecureSocksProxy?: boolean;
}

export interface DTSecureJsonData {
  apiKey?: string;
}
