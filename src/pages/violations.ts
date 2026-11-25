import { PanelBuilders, QueryVariable, SceneAppPage, SceneFlexItem, SceneFlexLayout } from '@grafana/scenes';
import {
  BigValueGraphMode,
  BigValueTextMode,
  FieldColorModeId,
  LineInterpolation,
  StackingMode,
} from '@grafana/schema';
import { QueryMode, QueryType } from '../datasource/types';
import { DS_PLUGIN_ID, LATEST, STATE_TEXT, colorText, pageScene, queries, stateColors, table } from '../scenes/common';
import { prefixRoute } from '../utils/utils.routing';
import { ROUTES } from '../constants';

function scene() {
  const policy = new QueryVariable({
    name: 'policy',
    label: 'Policy',
    datasource: { type: DS_PLUGIN_ID, uid: '${ds}' },
    query: { refId: 'policy', type: 'policies' },
    isMulti: true,
    includeAll: true,
    defaultToAll: true,
  });

  const stats = PanelBuilders.stat()
    .setTitle('Portfolio violations')
    .setDescription('Current portfolio-wide counts, independent of the policy filter.')
    .setData(
      queries({
        queryType: QueryType.PortfolioMetrics,
        mode: QueryMode.Current,
        metrics: [
          'policyViolationsFail',
          'policyViolationsWarn',
          'policyViolationsInfo',
          'policyViolationsUnaudited',
          'policyViolationsSecurityTotal',
          'policyViolationsLicenseTotal',
          'policyViolationsOperationalTotal',
        ],
      })
    )
    .setOption('graphMode', BigValueGraphMode.None)
    .setOption('textMode', BigValueTextMode.ValueAndName)
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'text' })
    .setOverrides(stateColors)
    .build();
  const trend = PanelBuilders.timeseries()
    .setTitle('Violations over time')
    .setData(
      queries({
        queryType: QueryType.PortfolioMetrics,
        mode: QueryMode.TimeSeries,
        metrics: ['policyViolationsFail', 'policyViolationsWarn', 'policyViolationsInfo'],
      })
    )
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'blue' })
    .setCustomFieldConfig('lineInterpolation', LineInterpolation.StepAfter)
    .setCustomFieldConfig('fillOpacity', 20)
    .setCustomFieldConfig('stacking', { mode: StackingMode.Normal, group: 'A' })
    .setOverrides(stateColors)
    .build();
  const list = table(
    'Violations',
    queries({
      queryType: QueryType.Violations,
      format: 'table',
      violations: { policies: ['$policy'] },
      projects: LATEST,
    }),
    ['Tags', 'PURL'],
    (b) => colorText(b, 'State', STATE_TEXT)
  ).build();

  return pageScene(
    new SceneFlexLayout({
      direction: 'column',
      children: [
        new SceneFlexItem({ height: 130, body: stats }),
        new SceneFlexItem({ minHeight: 260, body: trend }),
        new SceneFlexItem({ minHeight: 400, body: list }),
      ],
    }),
    [policy]
  );
}

export const violationsPage = new SceneAppPage({
  title: 'Policy violations',
  subTitle: 'Policy violations in the latest version of each project.',
  url: prefixRoute(ROUTES.Violations),
  routePath: ROUTES.Violations,
  getScene: scene,
});
