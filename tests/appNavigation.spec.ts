import { test, expect } from './fixtures';
import { ROUTES } from '../src/constants';

test.use({ viewport: { width: 1600, height: 2000 } });

const pages: Array<[ROUTES, string, string]> = [
  [ROUTES.Overview, 'Vulnerabilities by severity', 'inventory-service · critical'],
  [ROUTES.Projects, 'Projects', 'payments-api'],
  [ROUTES.Findings, 'Critical and high findings', 'INT-2021-0001'],
  [ROUTES.Violations, 'Violations', 'Banned packages'],
  [ROUTES.Freshness, 'SBOM freshness', 'CycloneDX 1.6'],
];

for (const [route, panel, content] of pages) {
  test(`${route} page renders data`, async ({ gotoPage, page }) => {
    await gotoPage(`/${route}`);
    await expect(page.getByText(panel, { exact: true }).first()).toBeVisible();
    await expect(page.getByText(content).first()).toBeVisible();
    await expect(page.getByTestId(/Panel status error/)).toHaveCount(0);
    await expect(page.getByRole('columnheader', { name: /UUID/ })).toHaveCount(0);
  });
}

test('project drill-down', async ({ gotoPage, page }) => {
  await gotoPage(`/${ROUTES.Projects}`);
  // The project cell offers both the in-app page and the backend's Dependency-Track link.
  await page.getByText('inventory-service', { exact: true }).first().click();
  await expect(page.getByRole('link', { name: 'Open project in Dependency-Track' })).toBeVisible();
  await page.getByRole('link', { name: 'Open project', exact: true }).click();
  await expect(page).toHaveURL(/\/projects\/inventory-service/);
  await expect(page.getByText('Vulnerabilities over time')).toBeVisible();

  await page.getByRole('tab', { name: 'Findings' }).click();
  await expect(page).toHaveURL(/\/projects\/inventory-service\/findings/);
  await expect(page.getByText('INT-2020-0008')).toBeVisible();
  await expect(page.getByText('INT-2021-0010')).toHaveCount(0);

  await page.getByRole('tab', { name: 'Versions' }).click();
  await expect(page.locator('a[href^="http"][href*="/projects/"]').first()).toHaveAttribute(
    'href',
    /\/projects\/[0-9a-f-]{36}$/
  );
  await expect(page.getByTestId(/Panel status error/)).toHaveCount(0);
});
