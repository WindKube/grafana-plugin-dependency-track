import { PanelBuilders, SceneAppPage, SceneFlexItem, SceneFlexLayout } from '@grafana/scenes';
import { BigValueGraphMode, BigValueTextMode, FieldColorModeId } from '@grafana/schema';
import { QueryMode, QueryType } from '../datasource/types';
import { LATEST, SEVERITY_TEXT, bars, colorText, pageScene, queries, table } from '../scenes/common';
import { prefixRoute } from '../utils/utils.routing';
import { ROUTES } from '../constants';

const HIDDEN = ['Tags', 'PURL', 'Attributed'];

function scene() {
  const stats = PanelBuilders.stat()
    .setTitle('Findings')
    .setDescription('Current portfolio-wide counts.')
    .setData(
      queries({
        queryType: QueryType.PortfolioMetrics,
        mode: QueryMode.Current,
        metrics: ['findingsTotal', 'findingsUnaudited', 'findingsAudited', 'suppressed', 'kev', 'vulnerableComponents'],
      })
    )
    .setOption('graphMode', BigValueGraphMode.None)
    .setOption('textMode', BigValueTextMode.ValueAndName)
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'text' })
    .build();
  const unaudited = bars('Unaudited findings by project')
    .setData(
      queries({
        queryType: QueryType.ProjectMetrics,
        mode: QueryMode.Current,
        metrics: ['findingsUnaudited'],
        projects: LATEST,
      })
    )
    .setDisplayName('${__field.labels.project}')
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'orange' })
    .build();
  const kev = table(
    'Known exploited vulnerabilities',
    queries({ queryType: QueryType.Findings, format: 'table', findings: { kevOnly: true }, projects: LATEST }),
    [...HIDDEN, 'KEV']
  )
    .setDescription('Findings listed in the CISA KEV catalog.')
    .setNoValue('No known exploited vulnerabilities')
    .build();
  const critical = table(
    'Critical and high findings',
    queries({
      queryType: QueryType.Findings,
      format: 'table',
      findings: { severities: ['CRITICAL', 'HIGH'] },
      projects: LATEST,
    }),
    HIDDEN,
    (b) => colorText(b, 'Severity', SEVERITY_TEXT)
  ).build();

  return pageScene(
    new SceneFlexLayout({
      direction: 'column',
      children: [
        new SceneFlexItem({ height: 130, body: stats }),
        new SceneFlexItem({ minHeight: 260, body: unaudited }),
        new SceneFlexItem({ minHeight: 200, body: kev }),
        new SceneFlexItem({ minHeight: 360, body: critical }),
      ],
    })
  );
}

export const findingsPage = new SceneAppPage({
  title: 'Findings',
  subTitle: 'Vulnerabilities found in the latest version of each project.',
  url: prefixRoute(ROUTES.Findings),
  routePath: ROUTES.Findings,
  getScene: scene,
});
