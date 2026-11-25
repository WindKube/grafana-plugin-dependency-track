import { DataSourceInstanceSettings, CoreApp, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { DTQuery, DTDataSourceOptions, DEFAULT_QUERY } from './types';

export class DataSource extends DataSourceWithBackend<DTQuery, DTDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<DTDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(_: CoreApp): Partial<DTQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: DTQuery, scopedVars: ScopedVars) {
    return {
      ...query,
      queryText: getTemplateSrv().replace(query.queryText, scopedVars),
    };
  }

  filterQuery(query: DTQuery): boolean {
    // if no query has been provided, prevent the query from being executed
    return !!query.queryText;
  }
}
