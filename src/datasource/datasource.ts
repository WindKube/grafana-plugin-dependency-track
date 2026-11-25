import { AnnotationQuery, CoreApp, DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { DEFAULT_QUERY, DTDataSourceOptions, DTQuery, ProjectOption, QueryType } from './types';
import { DTVariableSupport } from './variables';

const DEFAULT_ANNOTATION_QUERY: Partial<DTQuery> = {
  queryType: QueryType.Events,
  projects: { mode: 'all', onlyLatest: true },
  events: { kinds: ['bomImport', 'violation'] },
};

export class DataSource extends DataSourceWithBackend<DTQuery, DTDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<DTDataSourceOptions>) {
    super(instanceSettings);
    this.variables = new DTVariableSupport(this);
    this.annotations = {
      getDefaultQuery: () => DEFAULT_ANNOTATION_QUERY,
      // Grafana spreads getDefaultQuery() onto the annotation root, so an annotation saved before
      // its query was edited has no target and would never run. Return the same object when
      // nothing changes: the annotation editor compares by reference and loops otherwise.
      prepareAnnotation: (anno: AnnotationQuery<DTQuery>) =>
        anno.target ? anno : { ...anno, target: { refId: 'Anno', ...DEFAULT_ANNOTATION_QUERY } },
    };
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
