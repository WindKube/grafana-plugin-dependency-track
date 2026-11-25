import { PanelBuilders, SceneAppPage, SceneFlexItem, SceneFlexLayout } from '@grafana/scenes';
import { FieldColorModeId, GraphDrawStyle, TableCellDisplayMode, ThresholdsMode } from '@grafana/schema';
import { QueryMode, QueryType } from '../datasource/types';
import { LATEST, bars, neverImported, pageScene, queries, table } from '../scenes/common';
import { prefixRoute } from '../utils/utils.routing';
import { ROUTES } from '../constants';

const WEEK = 7 * 86400;
const MONTH = 30 * 86400;

function scene() {
  const freshness = table(
    'SBOM freshness',
    queries({ queryType: QueryType.Projects, format: 'table', projects: LATEST }),
    ['Active', 'Latest', 'Risk score', 'Critical', 'High', 'Medium', 'Low', 'Unassigned', 'Policy violations (fail)'],
    (b) => {
      neverImported(b);
      b.matchFieldsWithName('BOM age')
        .overrideCustomFieldConfig('cellOptions', { type: TableCellDisplayMode.ColorBackground })
        .overrideThresholds({
          mode: ThresholdsMode.Absolute,
          steps: [
            { color: 'green', value: -Infinity },
            { color: 'yellow', value: WEEK },
            { color: 'red', value: MONTH },
          ],
        });
    }
  )
    .setDescription('Green: imported in the last 7 days, yellow: in the last 30 days.')
    .build();
  const components = bars('Components by project')
    .setData(
      queries({
        queryType: QueryType.ProjectMetrics,
        mode: QueryMode.Current,
        metrics: ['components'],
        projects: LATEST,
      })
    )
    .setDisplayName('${__field.labels.project}')
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'blue' })
    .build();
  const database = PanelBuilders.timeseries()
    .setTitle('Vulnerability database')
    .setDescription(
      'Vulnerabilities published per month in the Dependency-Track mirror. Empty when no mirror is enabled.'
    )
    .setData(queries({ queryType: QueryType.VulnerabilityMetrics }))
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'purple' })
    .setCustomFieldConfig('drawStyle', GraphDrawStyle.Bars)
    .setCustomFieldConfig('fillOpacity', 60)
    .build();

  return pageScene(
    new SceneFlexLayout({
      direction: 'column',
      children: [
        new SceneFlexItem({ minHeight: 280, body: freshness }),
        new SceneFlexLayout({
          minHeight: 300,
          children: [new SceneFlexItem({ body: components }), new SceneFlexItem({ body: database })],
        }),
      ],
    })
  );
}

export const freshnessPage = new SceneAppPage({
  title: 'SBOM freshness',
  subTitle: 'When each project last received an SBOM.',
  url: prefixRoute(ROUTES.Freshness),
  routePath: ROUTES.Freshness,
  getScene: scene,
});
