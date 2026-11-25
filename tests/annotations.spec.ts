import { test, expect } from '@grafana/plugin-e2e';
import { APIRequestContext } from '@playwright/test';

type Field = { name: string; config?: { links?: Array<{ url: string }> } };
type Frame = { schema: { fields: Field[] }; data: { values: unknown[][] } };

async function dsQuery(request: APIRequestContext, query: Record<string, unknown>): Promise<Frame> {
  const res = await request.post('/api/ds/query', {
    data: {
      from: 'now-30d',
      to: 'now',
      queries: [{ refId: 'A', datasource: { uid: 'dependencytrack' }, ...query }],
    },
  });
  expect(res.ok()).toBeTruthy();
  return (await res.json()).results.A.frames[0];
}

test('events query returns annotation fields', async ({ request }) => {
  const frame = await dsQuery(request, {
    queryType: 'events',
    projects: { mode: 'all', onlyLatest: true },
  });
  expect(frame.schema.fields.map((f) => f.name)).toEqual(['time', 'title', 'text', 'tags', 'project', 'version']);
  const titles = frame.data.values[1] as string[];
  expect(titles.filter((t) => t.startsWith('BOM imported:'))).toHaveLength(4);
  expect(titles.filter((t) => t.startsWith('FAIL policy violation in web-frontend'))).toHaveLength(2);
  expect(titles.filter((t) => t.startsWith('CRITICAL '))).toHaveLength(2);
});

test('finding tables link to the Dependency-Track UI', async ({ request }) => {
  const frame = await dsQuery(request, { queryType: 'findings', format: 'table' });
  const project = frame.schema.fields.find((f) => f.name === 'Project');
  expect(project?.config?.links?.[0].url).toBe('http://localhost:8082/projects/${__data.fields["Project UUID"]}');
});

test('annotation editor defaults to an events query', async ({
  page,
  annotationEditPage,
  readProvisionedDataSource,
}) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await annotationEditPage.datasource.set(ds.name);
  await expect(page.getByText('BOM imports')).toBeVisible();
  const button = page.getByRole('button', { name: 'Test annotation query' });
  await expect(button).toBeVisible();
  const response = page.waitForResponse((r) => r.url().includes('/api/ds/query'));
  await button.click();
  await expect(await response).toBeOK();
  // 4 latest BOM imports + 8 policy violations from the seed.
  await expect(page.getByText(/^12 events/)).toBeVisible();
});
