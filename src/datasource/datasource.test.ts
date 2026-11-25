import { DataQueryRequest, DataSourceInstanceSettings } from '@grafana/data';
import { lastValueFrom } from 'rxjs';

import { DataSource } from './datasource';
import { DTVariableSupport } from './variables';
import { DTDataSourceOptions, DTQuery, DTVariableQuery, QueryType } from './types';

const variables: Record<string, string[]> = {
  team: ['payments-api', 'web-frontend'],
  tag: ['backend'],
  ids: ['uuid-1', 'uuid-2'],
};

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getTemplateSrv: () => ({
    getVariables: () => [],
    replace: (value: string, _scoped: unknown, format?: string) =>
      value.replace(/\$(\w+)/g, (_m: string, name: string) => {
        const values = variables[name] ?? [];
        if (format === 'regex') {
          return values.length > 1 ? `(${values.join('|')})` : values[0];
        }
        return format === 'csv' ? values.join(',') : values[0];
      }),
  }),
}));

function newDatasource() {
  return new DataSource({ jsonData: {} } as DataSourceInstanceSettings<DTDataSourceOptions>);
}

function runVariableQuery(ds: DataSource, query: DTVariableQuery) {
  const request = { targets: [query], scopedVars: {} } as unknown as DataQueryRequest<DTVariableQuery>;
  return lastValueFrom((ds.variables as DTVariableSupport).query(request));
}

describe('applyTemplateVariables', () => {
  it('interpolates project selectors', () => {
    const query: DTQuery = {
      refId: 'A',
      queryType: QueryType.ProjectMetrics,
      projects: { mode: 'name', nameRegex: '^$team$', tag: '$tag', uuids: ['$ids', 'uuid-3'] },
    };
    const out = newDatasource().applyTemplateVariables(query, {});
    expect(out.projects).toEqual({
      mode: 'name',
      nameRegex: '^(payments-api|web-frontend)$',
      tag: 'backend',
      classifier: undefined,
      uuids: ['uuid-1', 'uuid-2', 'uuid-3'],
    });
  });

  it('leaves portfolio queries alone', () => {
    const query: DTQuery = { refId: 'A', queryType: QueryType.PortfolioMetrics };
    expect(newDatasource().applyTemplateVariables(query, {})).toBe(query);
  });
});

describe('variable support', () => {
  it('returns distinct project names', async () => {
    const ds = newDatasource();
    jest.spyOn(ds, 'getProjects').mockResolvedValue([
      { uuid: '1', name: 'payments-api', version: '1.4.0', isLatest: false, tags: [] },
      { uuid: '2', name: 'payments-api', version: '1.5.0', isLatest: true, tags: [] },
      { uuid: '3', name: 'web-frontend', version: '2.3.0', isLatest: true, tags: [] },
    ]);
    const query: DTVariableQuery = { refId: 'A', type: 'projects' };
    const res = await runVariableQuery(ds, query);
    expect(res.data[0].fields[0].values).toEqual(['payments-api', 'web-frontend']);
  });

  it('interpolates the project of a versions query', async () => {
    const ds = newDatasource();
    const getVersions = jest.spyOn(ds, 'getVersions').mockResolvedValue(['1.4.0', '1.5.0']);
    const query: DTVariableQuery = { refId: 'A', type: 'versions', project: '$team' };
    const res = await runVariableQuery(ds, query);
    expect(getVersions).toHaveBeenCalledWith('payments-api');
    expect(res.data[0].fields[1].values).toEqual(['1.4.0', '1.5.0']);
  });
});
