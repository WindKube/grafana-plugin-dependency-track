import type { PluginOptions } from '@grafana/plugin-e2e';
import { defineConfig } from '@playwright/test';
import baseConfig from '../../.config/playwright.config';

export default defineConfig<PluginOptions>(baseConfig, {
  testDir: '.',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: 'line',
  use: {
    launchOptions: { args: ['--font-render-hinting=none'] },
  },
});
