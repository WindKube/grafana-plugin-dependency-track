import {
  DataSourceVariable,
  EmbeddedScene,
  FieldConfigOverridesBuilder,
  PanelBuilders,
  SceneControlsSpacer,
  SceneDataTransformer,
  SceneObject,
  SceneQueryRunner,
  SceneRefreshPicker,
  SceneTimePicker,
  SceneTimeRange,
  SceneVariable,
  SceneVariableSet,
  VariableValueSelectors,
} from '@grafana/scenes';
import { MappingType, SpecialValueMatch } from '@grafana/data';
import {
  BarGaugeDisplayMode,
  BarGaugeNamePlacement,
  FieldColorModeId,
  TableCellDisplayMode,
  TableCellHeight,
  TableFieldOptions,
  VizOrientation,
} from '@grafana/schema';
import { DTQuery, ProjectSelector } from '../datasource/types';
import { prefixRoute } from '../utils/utils.routing';
import { ROUTES } from '../constants';

export const DS_PLUGIN_ID = 'windkube-dependencytrack-datasource';
const DS = { type: DS_PLUGIN_ID, uid: '${ds}' };

export const LATEST: ProjectSelector = { mode: 'all', onlyLatest: true };

export function queries(...qs: Array<Partial<DTQuery>>): SceneQueryRunner {
  return new SceneQueryRunner({
    datasource: DS,
    queries: qs.map((q, i) => ({ refId: String.fromCharCode(65 + i), ...q })),
  });
}

export function pageScene(body: SceneObject, variables: SceneVariable[] = []): EmbeddedScene {
  return new EmbeddedScene({
    $timeRange: new SceneTimeRange({ from: 'now-90d', to: 'now' }),
    $variables: new SceneVariableSet({
      variables: [new DataSourceVariable({ name: 'ds', label: 'Data source', pluginId: DS_PLUGIN_ID }), ...variables],
    }),
    controls: [
      new VariableValueSelectors({}),
      new SceneControlsSpacer(),
      new SceneTimePicker({}),
      new SceneRefreshPicker({}),
    ],
    body,
  });
}

export function projectSelector(project: string): ProjectSelector {
  return { mode: 'name', nameRegex: `^${project.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')}$`, onlyLatest: true };
}

export function projectUrl(project: string): string {
  return `${prefixRoute(ROUTES.Projects)}/${encodeURIComponent(project)}`;
}

export const PROJECT_LINK = {
  title: 'Open project',
  url: `${prefixRoute(ROUTES.Projects)}/\${__field.labels.project:percentencode}?\${ds:queryparam}&\${__url_time_range}`,
};

const SEVERITY_COLORS: Array<[string, string]> = [
  ['Critical', 'dark-red'],
  ['High', 'orange'],
  ['Medium', 'yellow'],
  ['Low', 'blue'],
  ['Unassigned', ''],
];

// Colors series by the severity at the end of their display name. With shortNames, the
// "<project> <version>" prefix is dropped, for panels that show a single project.
export function severityColors<T extends {}>(b: FieldConfigOverridesBuilder<T>, shortNames = false) {
  for (const [severity, color] of SEVERITY_COLORS) {
    if (color) {
      b.matchFieldsWithNameByRegex(`(.* )?${severity}$`).overrideColor({
        mode: FieldColorModeId.Fixed,
        fixedColor: color,
      });
    }
    if (shortNames) {
      b.matchFieldsWithNameByRegex(`.* ${severity}$`).overrideDisplayName(severity);
    }
  }
}

// Bar gauges size their text to the panel, which makes it huge with only a few projects.
export function bars(title: string, displayMode = BarGaugeDisplayMode.Basic) {
  return PanelBuilders.bargauge()
    .setTitle(title)
    .setOption('orientation', VizOrientation.Horizontal)
    .setOption('displayMode', displayMode)
    .setOption('namePlacement', BarGaugeNamePlacement.Left)
    .setOption('text', { titleSize: 14, valueSize: 20 })
    .setMin(0)
    .setLinks([PROJECT_LINK]);
}

export function stateColors<T extends {}>(b: FieldConfigOverridesBuilder<T>) {
  b.matchFieldsWithName('Policy violations (fail)').overrideColor({ mode: FieldColorModeId.Fixed, fixedColor: 'red' });
  b.matchFieldsWithName('Policy violations (warn)').overrideColor({
    mode: FieldColorModeId.Fixed,
    fixedColor: 'orange',
  });
}

type TableOverrides = FieldConfigOverridesBuilder<TableFieldOptions>;

// Data links are interpolated against the transformed frame, so link columns are hidden, not excluded.
export function table(
  title: string,
  runner: SceneQueryRunner,
  exclude: string[] = [],
  overrides: (b: TableOverrides) => void = () => {}
) {
  return PanelBuilders.table()
    .setTitle(title)
    .setData(
      new SceneDataTransformer({
        $data: runner,
        transformations: [
          { id: 'organize', options: { excludeByName: Object.fromEntries(exclude.map((name) => [name, true])) } },
        ],
      })
    )
    .setOption('cellHeight', TableCellHeight.Sm)
    .setOverrides((b) => {
      // The table in Grafana 13.0 ignores hideFrom.viz and only honors the untyped "hidden".
      b.matchFieldsWithNameByRegex('.* UUID$').overrideCustomFieldConfig<
        TableFieldOptions & { hidden?: boolean },
        'hidden'
      >('hidden', true);
      overrides(b);
    });
}

export function colorText(b: TableOverrides, field: string, colors: Record<string, string>) {
  b.matchFieldsWithName(field)
    .overrideCustomFieldConfig('cellOptions', { type: TableCellDisplayMode.ColorText })
    .overrideMappings([
      {
        type: MappingType.ValueToText,
        options: Object.fromEntries(Object.entries(colors).map(([value, color], index) => [value, { color, index }])),
      },
    ]);
}

export const SEVERITY_TEXT = { CRITICAL: 'dark-red', HIGH: 'orange', MEDIUM: 'yellow', LOW: 'blue' };
export const STATE_TEXT = { FAIL: 'red', WARN: 'orange' };

export function neverImported(b: TableOverrides) {
  b.matchFieldsWithName('BOM age').overrideMappings([
    { type: MappingType.SpecialValue, options: { match: SpecialValueMatch.Null, result: { text: 'Never' } } },
  ]);
}
