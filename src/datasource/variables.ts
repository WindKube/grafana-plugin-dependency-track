import { from, map, Observable } from 'rxjs';
import {
  CustomVariableSupport,
  DataQueryRequest,
  DataQueryResponse,
  MetricFindValue,
  toDataFrame,
} from '@grafana/data';
import { getTemplateSrv } from '@grafana/runtime';

import type { DataSource } from './datasource';
import { VariableQueryEditor } from './components/VariableQueryEditor';
import { DTVariableQuery } from './types';

export class DTVariableSupport extends CustomVariableSupport<DataSource, DTVariableQuery> {
  editor = VariableQueryEditor;

  constructor(private readonly datasource: DataSource) {
    super();
  }

  query(request: DataQueryRequest<DTVariableQuery>): Observable<DataQueryResponse> {
    const query = request.targets[0];
    return from(this.findValues(query, request)).pipe(
      map((values) => ({
        data: [
          toDataFrame({
            fields: [
              { name: 'text', values: values.map((v) => v.text) },
              { name: 'value', values: values.map((v) => v.value ?? v.text) },
            ],
          }),
        ],
      }))
    );
  }

  async findValues(query: DTVariableQuery, request?: DataQueryRequest<DTVariableQuery>): Promise<MetricFindValue[]> {
    const ds = this.datasource;
    const names = (values: string[]) => values.map((text) => ({ text }));
    switch (query?.type) {
      case 'projects': {
        const projects = await ds.getProjects(query.onlyLatest);
        return [...new Set(projects.map((p) => p.name))].map((text) => ({ text }));
      }
      case 'versions': {
        const project = getTemplateSrv().replace(query.project ?? '', request?.scopedVars);
        return project ? names(await ds.getVersions(project)) : [];
      }
      case 'tags':
        return names(await ds.getTags());
      case 'classifiers':
        return names(await ds.getClassifiers());
      case 'policies':
        return names(await ds.getPolicies());
      default:
        return [];
    }
  }
}
