import { PanelBuilders, SceneAppPage, SceneFlexItem, SceneFlexLayout } from '@grafana/scenes';
import {
  BarGaugeDisplayMode,
  BigValueColorMode,
  BigValueGraphMode,
  BigValueTextMode,
  FieldColorModeId,
  LineInterpolation,
  StackingMode,
  ThresholdsMode,
} from '@grafana/schema';
import { DEFAULT_METRICS, QueryMode, QueryType } from '../datasource/types';
import { LATEST, bars, pageScene, queries, severityColors } from '../scenes/common';
import { prefixRoute } from '../utils/utils.routing';
import { ROUTES } from '../constants';

function severityStat() {
  return PanelBuilders.stat()
    .setTitle('Vulnerabilities by severity')
    .setDescription('Current portfolio-wide counts.')
    .setData(queries({ queryType: QueryType.PortfolioMetrics, mode: QueryMode.Current, metrics: DEFAULT_METRICS }))
    .setOption('colorMode', BigValueColorMode.Background)
    .setOption('graphMode', BigValueGraphMode.None)
    .setOption('textMode', BigValueTextMode.ValueAndName)
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'text' })
    .setOverrides((b) => severityColors(b))
    .build();
}

function portfolioStat() {
  return PanelBuilders.stat()
    .setTitle('Portfolio')
    .setData(
      queries({
        queryType: QueryType.PortfolioMetrics,
        mode: QueryMode.Current,
        metrics: ['projects', 'vulnerableProjects', 'kev', 'policyViolationsFail', 'inheritedRiskScore'],
      })
    )
    .setOption('graphMode', BigValueGraphMode.None)
    .setOption('textMode', BigValueTextMode.ValueAndName)
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'text' })
    .setOverrides((b) =>
      b
        .matchFieldsWithNameByRegex('Known exploited.*|Policy violations.*')
        .overrideColor({ mode: FieldColorModeId.Thresholds })
        .overrideThresholds({
          mode: ThresholdsMode.Absolute,
          steps: [
            { color: 'green', value: -Infinity },
            { color: 'red', value: 1 },
          ],
        })
    )
    .build();
}

function severityTrend() {
  return PanelBuilders.timeseries()
    .setTitle('Vulnerabilities over time')
    .setData(queries({ queryType: QueryType.PortfolioMetrics, mode: QueryMode.TimeSeries, metrics: DEFAULT_METRICS }))
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'text' })
    .setCustomFieldConfig('lineInterpolation', LineInterpolation.StepAfter)
    .setCustomFieldConfig('fillOpacity', 20)
    .setCustomFieldConfig('stacking', { mode: StackingMode.Normal, group: 'A' })
    .setOverrides((b) => severityColors(b))
    .build();
}

function criticalByProject() {
  return bars('Critical and high by project')
    .setDescription('Latest version of each project. Click a bar to open the project.')
    .setData(
      queries({
        queryType: QueryType.ProjectMetrics,
        mode: QueryMode.Current,
        metrics: ['critical', 'high'],
        projects: LATEST,
      })
    )
    .setOverrides((b) => {
      severityColors(b);
      b.matchFieldsWithNameByRegex('.* Critical$').overrideDisplayName('${__field.labels.project} · critical');
      b.matchFieldsWithNameByRegex('.* High$').overrideDisplayName('${__field.labels.project} · high');
    })
    .build();
}

function riskByProject() {
  return bars('Inherited risk score by project', BarGaugeDisplayMode.Gradient)
    .setData(
      queries({
        queryType: QueryType.ProjectMetrics,
        mode: QueryMode.Current,
        metrics: ['inheritedRiskScore'],
        projects: LATEST,
      })
    )
    .setDisplayName('${__field.labels.project}')
    .setColor({ mode: FieldColorModeId.ContinuousGrYlRd })
    .build();
}

export const overviewPage = new SceneAppPage({
  title: 'Overview',
  subTitle: 'Software supply chain risk from OWASP Dependency-Track.',
  url: prefixRoute(ROUTES.Overview),
  routePath: ROUTES.Overview,
  getScene: () =>
    pageScene(
      new SceneFlexLayout({
        direction: 'column',
        children: [
          new SceneFlexLayout({
            height: 160,
            children: [new SceneFlexItem({ body: severityStat() }), new SceneFlexItem({ body: portfolioStat() })],
          }),
          new SceneFlexItem({ height: 280, body: severityTrend() }),
          new SceneFlexLayout({
            minHeight: 320,
            children: [new SceneFlexItem({ body: criticalByProject() }), new SceneFlexItem({ body: riskByProject() })],
          }),
        ],
      })
    ),
});
