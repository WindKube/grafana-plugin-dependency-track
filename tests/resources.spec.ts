import { test, expect } from '@grafana/plugin-e2e';

const base = '/api/datasources/uid/dependencytrack/resources';

test('resource routes return seeded Dependency-Track data', async ({ request }) => {
  const tags = await (await request.get(`${base}/tags`)).json();
  expect(tags).toEqual(expect.arrayContaining(['backend', 'production', 'team-payments']));

  const projects = await (await request.get(`${base}/projects?onlyLatest=true`)).json();
  expect(projects.map((p: { name: string }) => p.name)).toEqual([
    'inventory-service',
    'payments-api',
    'report-worker',
    'web-frontend',
  ]);

  const versions = await (await request.get(`${base}/versions?project=payments-api`)).json();
  expect(versions).toEqual(['1.4.0', '1.5.0']);

  const policies = await (await request.get(`${base}/policies`)).json();
  expect(policies).toEqual(['Banned packages', 'Embedded databases', 'Python review']);
});
