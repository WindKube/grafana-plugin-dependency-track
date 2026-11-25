import pluginJson from './plugin.json';

export const PLUGIN_BASE_URL = `/a/${pluginJson.id}`;

export const DATASOURCE_TYPE = 'windkube-dependencytrack-datasource';

export enum ROUTES {
  Home = 'home',
}
