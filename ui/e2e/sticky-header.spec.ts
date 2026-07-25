import { test, expect } from '@playwright/test';

test.describe('Sticky Header Verification Harness', () => {
  test.beforeEach(async ({ page }) => {
    // Mock category events response to have enough scrollable height
    await page.route('/api/v1/events*', async route => {
      const mockEvents = [];
      for (let i = 1; i <= 30; i++) {
        mockEvents.push({
          eventId: `BGG${i}`,
          anchorEventId: `BGG${i}`,
          title: `Sample Event ${i}`,
          shortDescription: `Short description for sample event ${i}`,
          categoryCode: 'BGG',
          categoryName: 'Board Games',
          systemName: 'Board Game System',
          year: 2026,
          wedTickets: 5,
          thuTickets: 5,
          friTickets: 5,
          satTickets: 5,
          sunTickets: 5
        });
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockEvents)
      });
    });

    await page.route('/api/v1/user', async route => {
      await route.fulfill({ status: 200, json: null });
    });

    await page.route('/api/v1/user/parties', async route => {
      await route.fulfill({ status: 200, json: [] });
    });
  });

  test('Desktop: Navbar remains visible and fixed on scroll down', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/cat/2026/BGG');
    await page.waitForSelector('.deferred-catalog-group');

    const navbar = page.locator('app-navbar nav');
    await expect(navbar).toBeVisible();

    // Scroll down 400px
    await page.evaluate(() => window.scrollTo(0, 400));
    await page.waitForTimeout(300);

    // Navbar should still be visible and not hidden
    const isHiddenOnBody = await page.evaluate(() => document.body.classList.contains('nav-hidden'));
    expect(isHiddenOnBody).toBe(false);
    await expect(navbar).toBeInViewport();
  });

  test('Mobile: Navbar hides on scroll down and re-appears on scroll up', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/cat/2026/BGG');
    await page.waitForSelector('.deferred-catalog-group');

    // Scroll down 300px to trigger hide
    await page.evaluate(() => window.scrollTo(0, 300));
    await page.waitForTimeout(300);

    let isHiddenOnBody = await page.evaluate(() => document.body.classList.contains('nav-hidden'));
    expect(isHiddenOnBody).toBe(true);

    // Verify sub-header top offset adjusts to 0px when navbar hides
    const stickyTop = await page.evaluate(() => {
      const elem = document.querySelector('.sticky-top-dynamic');
      return elem ? getComputedStyle(elem).top : '';
    });
    expect(stickyTop).toBe('0px');

    // Scroll up 100px to trigger re-appear
    await page.evaluate(() => window.scrollTo(0, 150));
    await page.waitForTimeout(300);

    isHiddenOnBody = await page.evaluate(() => document.body.classList.contains('nav-hidden'));
    expect(isHiddenOnBody).toBe(false);
  });
});
