import { join } from 'node:path';
import type { Page } from '@playwright/test';
import { test, expect } from '../../tests/fixtures';
import { ROUTES } from '../../src/constants';

const OUT = join(__dirname, '../../src/img/screenshots');
const DS_UID = 'dependencytrack';

test.use({ viewport: { width: 1600, height: 1000 } });

async function settle(page: Page) {
  await page.waitForLoadState('networkidle');
  await expect(page.getByLabel('Panel loading bar')).toHaveCount(0);
  await expect(page.getByTestId(/Panel status error/)).toHaveCount(0);
  await page.mouse.move(0, 0);
}

async function shoot(page: Page, name: string) {
  await settle(page);
  await page.screenshot({ path: join(OUT, `${name}.png`) });
}

const appPages: Array<[string, string]> = [
  [ROUTES.Overview, 'app-overview'],
  [ROUTES.Projects, 'app-projects'],
  [`${ROUTES.Projects}/inventory-service`, 'app-project'],
  [ROUTES.Findings, 'app-findings'],
  [ROUTES.Violations, 'app-violations'],
  [ROUTES.Freshness, 'app-freshness'],
];

for (const [route, name] of appPages) {
  test(name, async ({ gotoPage, page }) => {
    await gotoPage(`/${route}`);
    await shoot(page, name);
  });
}

const dashboards = ['portfolio-overview', 'project', 'findings', 'policy-violations', 'sbom-freshness'];

for (const uid of dashboards) {
  test(`dashboard-${uid}`, async ({ gotoDashboardPage, page }) => {
    await gotoDashboardPage({ uid: `dt-${uid}` });
    await shoot(page, `dashboard-${uid}`);
  });
}

test('datasource-config', async ({ gotoDataSourceConfigPage, page }) => {
  const configPage = await gotoDataSourceConfigPage(DS_UID);
  await expect(configPage.saveAndTest()).toBeOK();
  await expect(configPage).toHaveAlert('success');
  await page.getByText('Save & test').scrollIntoViewIfNeeded();
  await shoot(page, 'datasource-config');
});

test('query-editor', async ({ panelEditPage, page }) => {
  await panelEditPage.datasource.set('Dependency-Track');
  await panelEditPage.setVisualization('Bar gauge');
  await expect(panelEditPage.refreshPanel()).toBeOK();
  await shoot(page, 'query-editor');
});

test('alert-rule', async ({ alertRuleEditPage, page }) => {
  await alertRuleEditPage.alertRuleNameField.fill('Critical findings');
  await expect(alertRuleEditPage.evaluate()).toBeOK();
  await shoot(page, 'alert-rule');
});
