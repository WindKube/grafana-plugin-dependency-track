import { CoreApp, DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';

import { DEFAULT_QUERY, DTDataSourceOptions, DTQuery } from './types';

export class DataSource extends DataSourceWithBackend<DTQuery, DTDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<DTDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(_: CoreApp): Partial<DTQuery> {
    return DEFAULT_QUERY;
  }
}
