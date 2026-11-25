import { CoreApp, DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { DEFAULT_QUERY, DTDataSourceOptions, DTQuery, ProjectOption } from './types';
import { DTVariableSupport } from './variables';

export class DataSource extends DataSourceWithBackend<DTQuery, DTDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<DTDataSourceOptions>) {
    super(instanceSettings);
    this.variables = new DTVariableSupport(this);
  }

  getDefaultQuery(_: CoreApp): Partial<DTQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: DTQuery, scopedVars: ScopedVars): DTQuery {
    const { projects, violations } = query;
    if (!projects && !violations) {
      return query;
    }
    const srv = getTemplateSrv();
    const replace = (value?: string, format?: string) => (value ? srv.replace(value, scopedVars, format) : value);
    const expand = (values?: string[]) => values?.flatMap((v) => (replace(v, 'csv') ?? '').split(',')).filter(Boolean);
    return {
      ...query,
      projects: projects && {
        ...projects,
        nameRegex: replace(projects.nameRegex, 'regex'),
        tag: replace(projects.tag),
        classifier: replace(projects.classifier),
        uuids: expand(projects.uuids),
      },
      violations: violations && { ...violations, policies: expand(violations.policies) },
    };
  }

  getProjects(onlyLatest = false): Promise<ProjectOption[]> {
    return this.getResource('projects', onlyLatest ? { onlyLatest: 'true' } : {});
  }

  getVersions(project: string): Promise<string[]> {
    return this.getResource('versions', { project });
  }

  getTags(): Promise<string[]> {
    return this.getResource('tags');
  }

  getClassifiers(): Promise<string[]> {
    return this.getResource('classifiers');
  }

  getPolicies(): Promise<string[]> {
    return this.getResource('policies');
  }
}
