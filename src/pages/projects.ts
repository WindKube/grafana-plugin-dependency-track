import {
  PanelBuilders,
  SceneAppPage,
  SceneAppPageLike,
  SceneFlexItem,
  SceneFlexLayout,
  SceneRouteMatch,
} from '@grafana/scenes';
import {
  BigValueColorMode,
  BigValueGraphMode,
  BigValueTextMode,
  FieldColorModeId,
  LineInterpolation,
  StackingMode,
} from '@grafana/schema';
import { DEFAULT_METRICS, ProjectSelector, QueryMode, QueryType } from '../datasource/types';
import {
  LATEST,
  SEVERITY_TEXT,
  STATE_TEXT,
  colorText,
  neverImported,
  pageScene,
  projectSelector,
  projectUrl,
  queries,
  severityColors,
  table,
} from '../scenes/common';
import { prefixRoute } from '../utils/utils.routing';
import { ROUTES } from '../constants';

const PROJECT_PAGE_LINK = {
  title: 'Open project',
  url: `${prefixRoute(ROUTES.Projects)}/\${__value.raw:percentencode}?\${ds:queryparam}&\${__url_time_range}`,
};

function projectList() {
  return table(
    'Projects',
    queries({ queryType: QueryType.Projects, format: 'table', projects: LATEST }),
    ['Active', 'Latest', 'BOM format'],
    (b) => {
      b.matchFieldsWithName('Project').overrideLinks([PROJECT_PAGE_LINK]);
      neverImported(b);
    }
  ).build();
}

function projectOverview(sel: ProjectSelector) {
  const stat = PanelBuilders.stat()
    .setTitle('Vulnerabilities')
    .setData(
      queries({ queryType: QueryType.ProjectMetrics, mode: QueryMode.Current, metrics: DEFAULT_METRICS, projects: sel })
    )
    .setOption('colorMode', BigValueColorMode.Background)
    .setOption('graphMode', BigValueGraphMode.None)
    .setOption('textMode', BigValueTextMode.ValueAndName)
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'text' })
    .setOverrides((b) => severityColors(b, true))
    .build();
  const violations = PanelBuilders.stat()
    .setTitle('Policy violations')
    .setData(
      queries({
        queryType: QueryType.ProjectMetrics,
        mode: QueryMode.Current,
        metrics: ['policyViolationsFail', 'policyViolationsWarn', 'policyViolationsInfo'],
        projects: sel,
      })
    )
    .setOption('graphMode', BigValueGraphMode.None)
    .setOption('textMode', BigValueTextMode.ValueAndName)
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'text' })
    .setOverrides((b) => {
      b.matchFieldsWithNameByRegex('.*\\(fail\\)$')
        .overrideDisplayName('Fail')
        .overrideColor({ mode: FieldColorModeId.Fixed, fixedColor: 'red' });
      b.matchFieldsWithNameByRegex('.*\\(warn\\)$')
        .overrideDisplayName('Warn')
        .overrideColor({ mode: FieldColorModeId.Fixed, fixedColor: 'orange' });
      b.matchFieldsWithNameByRegex('.*\\(info\\)$').overrideDisplayName('Info');
    })
    .build();
  const trend = PanelBuilders.timeseries()
    .setTitle('Vulnerabilities over time')
    .setData(
      queries({
        queryType: QueryType.ProjectMetrics,
        mode: QueryMode.TimeSeries,
        metrics: DEFAULT_METRICS,
        projects: sel,
      })
    )
    .setColor({ mode: FieldColorModeId.Fixed, fixedColor: 'text' })
    .setCustomFieldConfig('lineInterpolation', LineInterpolation.StepAfter)
    .setCustomFieldConfig('fillOpacity', 20)
    .setCustomFieldConfig('stacking', { mode: StackingMode.Normal, group: 'A' })
    .setOverrides((b) => severityColors(b, true))
    .build();

  return new SceneFlexLayout({
    direction: 'column',
    children: [
      new SceneFlexLayout({
        height: 160,
        children: [new SceneFlexItem({ width: '62%', body: stat }), new SceneFlexItem({ body: violations })],
      }),
      new SceneFlexItem({ height: 320, body: trend }),
    ],
  });
}

function findings(sel: ProjectSelector) {
  return new SceneFlexItem({
    minHeight: 400,
    body: table(
      'Findings',
      queries({ queryType: QueryType.Findings, format: 'table', projects: sel }),
      ['Project', 'Version', 'Tags', 'PURL', 'Attributed'],
      (b) => colorText(b, 'Severity', SEVERITY_TEXT)
    ).build(),
  });
}

function violations(sel: ProjectSelector) {
  return new SceneFlexItem({
    minHeight: 400,
    body: table(
      'Policy violations',
      queries({ queryType: QueryType.Violations, format: 'table', projects: sel }),
      ['Project', 'Version', 'Tags', 'PURL'],
      (b) => colorText(b, 'State', STATE_TEXT)
    ).build(),
  });
}

function versions(project: string) {
  return new SceneFlexItem({
    minHeight: 300,
    body: table(
      'Versions',
      queries({
        queryType: QueryType.Projects,
        format: 'table',
        projects: { ...projectSelector(project), onlyLatest: false, includeInactive: true },
      }),
      ['Tags', 'Classifier'],
      neverImported
    )
      .setDescription('Click the project name to open this version in Dependency-Track.')
      .build(),
  });
}

function column(child: SceneFlexItem) {
  return new SceneFlexLayout({ direction: 'column', children: [child] });
}

function projectPage(routeMatch: SceneRouteMatch<{ project?: string }>, parent: SceneAppPageLike) {
  const project = routeMatch.params.project ?? '';
  const base = projectUrl(project);
  const sel = projectSelector(project);
  return new SceneAppPage({
    title: project,
    subTitle: 'Latest version of the project.',
    url: base,
    routePath: ':project/*',
    getParentPage: () => parent,
    preserveUrlKeys: ['var-ds', 'from', 'to'],
    tabs: [
      new SceneAppPage({
        title: 'Overview',
        url: base,
        routePath: '',
        getScene: () => pageScene(projectOverview(sel)),
      }),
      new SceneAppPage({
        title: 'Findings',
        url: `${base}/findings`,
        routePath: 'findings',
        getScene: () => pageScene(column(findings(sel))),
      }),
      new SceneAppPage({
        title: 'Policy violations',
        url: `${base}/violations`,
        routePath: 'violations',
        getScene: () => pageScene(column(violations(sel))),
      }),
      new SceneAppPage({
        title: 'Versions',
        url: `${base}/versions`,
        routePath: 'versions',
        getScene: () => pageScene(column(versions(project))),
      }),
    ],
  });
}

export const projectsPage = new SceneAppPage({
  title: 'Projects',
  subTitle: 'Latest version of every active project. Select a project for details.',
  url: prefixRoute(ROUTES.Projects),
  routePath: `${ROUTES.Projects}/*`,
  preserveUrlKeys: ['var-ds', 'from', 'to'],
  getScene: () => pageScene(column(new SceneFlexItem({ minHeight: 500, body: projectList() }))),
  drilldowns: [{ routePath: ':project/*', getPage: projectPage }],
});
