import pluginJson from './plugin.json';

export const PLUGIN_BASE_URL = `/a/${pluginJson.id}`;

export enum ROUTES {
  Overview = 'overview',
  Projects = 'projects',
  Findings = 'findings',
  Violations = 'violations',
  Freshness = 'freshness',
}
