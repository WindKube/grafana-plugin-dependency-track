import { SceneAppPage } from '@grafana/scenes';
import { homeScene } from './homeScene';
import { prefixRoute } from '../../utils/utils.routing';
import { ROUTES } from '../../constants';

export const homePage = new SceneAppPage({
  title: 'Dependency-Track',
  url: prefixRoute(ROUTES.Home),
  routePath: ROUTES.Home,
  subTitle: 'Software supply chain risk from OWASP Dependency-Track.',
  getScene: () => homeScene(),
});
