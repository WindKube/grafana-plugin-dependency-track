import { test, expect } from '@grafana/plugin-e2e';
import pluginJson from '../src/plugin.json';
import portfolio from '../src/dashboards/portfolio-overview.json';
import project from '../src/dashboards/project.json';
import findings from '../src/dashboards/findings.json';
import policy from '../src/dashboards/policy-violations.json';
import freshness from '../src/dashboards/sbom-freshness.json';

type Target = Record<string, unknown> & { refId: string };
type Transformation = { id: string; options?: { excludeByName?: object; includeByName?: object } };
type Dashboard = {
  uid: string;
  panels: Array<{ title: string; targets: Target[]; transformations?: Transformation[] }>;
  annotations: { list: Array<{ name: string; target?: Target }> };
};

const dashboards: Record<string, Dashboard> = {
  'dashboards/portfolio-overview.json': portfolio,
  'dashboards/project.json': project,
  'dashboards/findings.json': findings,
  'dashboards/policy-violations.json': policy,
  'dashboards/sbom-freshness.json': freshness,
};

// Stand-in for template interpolation: every project, every policy.
function interpolate(target: Target, dsUid: string): Target {
  const json = JSON.stringify(target).replaceAll('["$policy"]', '[]').replaceAll('$project', '.*');
  return { ...JSON.parse(json), datasource: { uid: dsUid } };
}

test('every bundled dashboard is listed in plugin.json', () => {
  const included = pluginJson.includes.filter((i) => i.type === 'dashboard').map((i) => i.path);
  expect(included.sort()).toEqual(Object.keys(dashboards).sort());
});

// Data links interpolate the transformed frame, so dropping a column breaks the links that use it.
// Hide such columns with a custom.hidden override instead.
test('tables keep the columns data links need', () => {
  for (const [path, dashboard] of Object.entries(dashboards)) {
    for (const panel of dashboard.panels) {
      for (const t of panel.transformations ?? []) {
        const where = `${path}: ${panel.title}`;
        expect(t.options?.includeByName, where).toBeUndefined();
        expect(
          Object.keys(t.options?.excludeByName ?? {}).filter((n) => /UUID$|^Source$/.test(n)),
          where
        ).toEqual([]);
      }
    }
  }
});

for (const [path, dashboard] of Object.entries(dashboards)) {
  test(`${path} is imported and every query succeeds`, async ({ request, readProvisionedDataSource }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    const imported = await request.get(`/api/dashboards/uid/${dashboard.uid}`);
    expect(imported.ok()).toBeTruthy();

    const targets = [
      ...dashboard.panels.flatMap((p) => p.targets.map((t) => ({ name: p.title, target: t }))),
      ...dashboard.annotations.list.filter((a) => a.target).map((a) => ({ name: a.name, target: a.target! })),
    ];
    for (const { name, target } of targets) {
      const res = await request.post('/api/ds/query', {
        data: { from: 'now-90d', to: 'now', queries: [interpolate(target, ds.uid)] },
      });
      expect(res.ok(), `${name}: ${await res.text()}`).toBeTruthy();
      const result = (await res.json()).results[target.refId];
      expect(result.error, name).toBeUndefined();
      expect(result.frames?.length, name).toBeGreaterThan(0);
    }
  });
}

test.describe('rendering', () => {
  // Panels below the fold are lazy-loaded and never mount in a normal viewport.
  test.use({ viewport: { width: 1600, height: 2400 } });

  test('portfolio overview renders without panel errors', async ({ gotoDashboardPage, page }) => {
    const dashboardPage = await gotoDashboardPage({ uid: portfolio.uid });
    for (const panel of portfolio.panels) {
      await expect(dashboardPage.getPanelByTitle(panel.title).locator).toBeVisible();
      await expect(dashboardPage.getPanelByTitle(panel.title).getErrorIcon()).toHaveCount(0);
    }
    await expect(page.getByText('inventory-service').first()).toBeVisible();
    await expect(page.locator('a[href*="/projects/"]').first()).toHaveAttribute('href', /\/projects\/[0-9a-f-]{36}$/);
    await expect(page.getByRole('columnheader', { name: /UUID/ })).toHaveCount(0);
  });
});
