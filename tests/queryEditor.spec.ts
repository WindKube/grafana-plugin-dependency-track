import { test, expect } from '@grafana/plugin-e2e';

test('smoke: should render query editor', async ({ page, panelEditPage, readProvisionedDataSource }) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await panelEditPage.datasource.set(ds.name);
  // Combobox in Grafana < 13.2 does not expose aria-label as its accessible name.
  await expect(page.getByText('Query type', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Remove Critical' })).toBeVisible();
});

test('portfolio metrics return the seeded critical count', async ({ panelEditPage, readProvisionedDataSource }) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await panelEditPage.datasource.set(ds.name);
  await panelEditPage.setVisualization('Table');
  await expect(panelEditPage.refreshPanel()).toBeOK();
  await expect(panelEditPage.panel.data).toContainText(['3']);
});
