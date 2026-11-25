import { test, expect } from './fixtures';
import { ROUTES } from '../src/constants';

test('home page renders', async ({ gotoPage, page }) => {
  await gotoPage(`/${ROUTES.Home}`);
  await expect(page.getByText('Software supply chain risk from OWASP Dependency-Track.')).toBeVisible();
});
