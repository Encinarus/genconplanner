import { test, expect } from '@playwright/test';

test.describe('Screen Edge Padding Verification', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('/api/v1/events*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            eventId: 'BGG1',
            anchorEventId: 'BGG1',
            title: 'Sample Event 1',
            shortDescription: 'Short description for sample event 1',
            categoryCode: 'BGG',
            categoryName: 'Board Games',
            systemName: 'Board Game System',
            year: 2026,
            wedTickets: 5,
            thuTickets: 5,
            friTickets: 5,
            satTickets: 5,
            sunTickets: 5
          }
        ])
      });
    });

    await page.route('/api/v1/user', async route => {
      await route.fulfill({ status: 200, json: null });
    });

    await page.route('/api/v1/user/parties', async route => {
      await route.fulfill({ status: 200, json: [] });
    });
  });

  test('Mobile: Has at least 12px screen edge padding and no horizontal overflow', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/cat/2026/BGG');
    await page.waitForSelector('.main');

    const paddingLeft = await page.evaluate(() => {
      const mainElem = document.querySelector('.main');
      return mainElem ? getComputedStyle(mainElem).paddingLeft : '0px';
    });

    const paddingRight = await page.evaluate(() => {
      const mainElem = document.querySelector('.main');
      return mainElem ? getComputedStyle(mainElem).paddingRight : '0px';
    });

    expect(parseInt(paddingLeft, 10)).toBeGreaterThanOrEqual(12);
    expect(parseInt(paddingRight, 10)).toBeGreaterThanOrEqual(12);

    const hasHorizontalOverflow = await page.evaluate(() => document.body.scrollWidth > window.innerWidth);
    expect(hasHorizontalOverflow).toBe(false);
  });

  test('Desktop: Retains desktop side padding', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/cat/2026/BGG');
    await page.waitForSelector('.main');

    const paddingLeft = await page.evaluate(() => {
      const mainElem = document.querySelector('.main');
      return mainElem ? getComputedStyle(mainElem).paddingLeft : '0px';
    });

    expect(paddingLeft).toBe('40px');
  });
});
