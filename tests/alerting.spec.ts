import { test, expect } from '@grafana/plugin-e2e';

const DS_UID = 'dependencytrack';

function rule(model: Record<string, unknown>, threshold = 0) {
  return {
    condition: 'C',
    data: [
      { refId: 'A', datasourceUid: DS_UID, relativeTimeRange: { from: 600, to: 0 }, model: { refId: 'A', ...model } },
      {
        refId: 'B',
        datasourceUid: '__expr__',
        relativeTimeRange: { from: 0, to: 0 },
        model: { refId: 'B', type: 'reduce', expression: 'A', reducer: 'last' },
      },
      {
        refId: 'C',
        datasourceUid: '__expr__',
        relativeTimeRange: { from: 0, to: 0 },
        model: {
          refId: 'C',
          type: 'threshold',
          expression: 'B',
          conditions: [{ evaluator: { type: 'gt', params: [threshold] } }],
        },
      },
    ],
  };
}

type Frame = {
  schema: { fields: Array<{ type: string; labels?: Record<string, string> }> };
  data: { values: unknown[][] };
};

function series(frames: Frame[]) {
  return frames.map((f) => {
    const idx = f.schema.fields.findIndex((field) => field.type === 'number');
    return { labels: f.schema.fields[idx].labels ?? {}, value: f.data.values[idx][0] as number };
  });
}

test('alert rule on critical findings fires per project', async ({ request }) => {
  const res = await request.post('/api/v1/eval', {
    data: rule({
      queryType: 'projectMetrics',
      mode: 'current',
      metrics: ['critical'],
      projects: { mode: 'all', onlyLatest: true },
    }),
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  const results = series(body.results.C.frames);

  const firing = results.filter((r) => r.value === 1).map((r) => r.labels.project);
  expect(firing.sort()).toEqual(['inventory-service']);
  expect(results).toHaveLength(4);
  for (const r of results) {
    expect(r.labels).toMatchObject({ metric: 'critical', severity: 'critical' });
    expect(r.labels.version).toBeTruthy();
  }
});

test('alert rule on portfolio policy violations', async ({ request }) => {
  const res = await request.post('/api/v1/eval', {
    data: rule({ queryType: 'portfolioMetrics', mode: 'current', metrics: ['policyViolationsFail', 'kev'] }),
  });
  expect(res.ok()).toBeTruthy();
  const results = series((await res.json()).results.C.frames);
  expect(results).toHaveLength(2);
  expect(results.find((r) => r.labels.metric === 'policyViolationsFail')?.value).toBe(1);
  expect(results.find((r) => r.labels.metric === 'kev')?.value).toBe(0);
});

test('alert rule on FAIL policy violations counts per project', async ({ request }) => {
  const res = await request.post('/api/v1/eval', {
    data: rule({
      queryType: 'violations',
      format: 'count',
      violations: { states: ['FAIL'], riskTypes: ['OPERATIONAL'] },
    }),
  });
  expect(res.ok()).toBeTruthy();
  const results = series((await res.json()).results.C.frames);
  expect(results).toHaveLength(5);
  const firing = results.filter((r) => r.value === 1);
  expect(firing).toHaveLength(1);
  expect(firing[0].labels).toMatchObject({ project: 'web-frontend', state: 'FAIL', type: 'OPERATIONAL' });
});

test('alert rule on stale SBOMs', async ({ request }) => {
  const query = { queryType: 'projects', format: 'bomAge', projects: { mode: 'all', onlyLatest: true } };
  const evaluate = async (threshold: number) => {
    const res = await request.post('/api/v1/eval', { data: rule(query, threshold) });
    expect(res.ok()).toBeTruthy();
    return series((await res.json()).results.C.frames);
  };

  const fresh = await evaluate(0);
  expect(fresh.map((r) => r.labels.project).sort()).toEqual([
    'inventory-service',
    'payments-api',
    'report-worker',
    'web-frontend',
  ]);
  expect(fresh.every((r) => r.value === 1)).toBeTruthy();

  const stale = await evaluate(30 * 24 * 3600);
  expect(stale.map((r) => r.labels.project).sort()).toEqual(fresh.map((r) => r.labels.project).sort());
  expect(stale.every((r) => r.value === 0)).toBeTruthy();
});

test('alert rule on critical findings count', async ({ request }) => {
  const res = await request.post('/api/v1/eval', {
    data: rule({
      queryType: 'findings',
      format: 'count',
      findings: { severities: ['CRITICAL'] },
      projects: { mode: 'all', onlyLatest: true },
    }),
  });
  expect(res.ok()).toBeTruthy();
  const results = series((await res.json()).results.C.frames);
  expect(results.filter((r) => r.value === 1).map((r) => r.labels.project)).toEqual(['inventory-service']);
  expect(results.every((r) => r.labels.severity === 'critical')).toBeTruthy();
});
