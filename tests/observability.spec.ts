import { test, expect } from '@grafana/plugin-e2e';
import { APIRequestContext } from '@playwright/test';

async function metric(request: APIRequestContext, name: string, labels: string): Promise<number> {
  const res = await request.get('/api/plugins/windkube-dependencytrack-datasource/metrics');
  expect(res.ok()).toBeTruthy();
  const line = (await res.text()).split('\n').find((l) => l.startsWith(`${name}{${labels}}`));
  return line ? Number(line.split(' ').pop()) : 0;
}

test('repeated queries are served from the cache and exposed as metrics', async ({ request }) => {
  const query = {
    from: 'now-1h',
    to: 'now',
    queries: [{ refId: 'A', datasource: { uid: 'dependencytrack' }, queryType: 'projects' }],
  };
  expect((await request.post('/api/ds/query', { data: query })).ok()).toBeTruthy();
  const hitsBefore = await metric(request, 'dependencytrack_cache_requests_total', 'result="hit"');

  expect((await request.post('/api/ds/query', { data: query })).ok()).toBeTruthy();
  expect(await metric(request, 'dependencytrack_cache_requests_total', 'result="hit"')).toBeGreaterThan(hitsBefore);

  expect(
    await metric(request, 'dependencytrack_api_request_duration_seconds_count', 'endpoint="/v1/project",status="200"')
  ).toBeGreaterThan(0);
  expect(
    await metric(request, 'dependencytrack_query_duration_seconds_count', 'query_type="projects",status="ok"')
  ).toBeGreaterThanOrEqual(2);
});
