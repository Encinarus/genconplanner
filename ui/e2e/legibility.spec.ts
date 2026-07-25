import { test, expect } from '@playwright/test';

test.describe('Mobile Legibility & Small Text Verification', () => {
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
            wedTickets: 0,
            thuTickets: 0,
            friTickets: 0,
            satTickets: 0,
            sunTickets: 0
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

  test('Mobile: Small text elements resolve to at least 12px font-size', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/cat/2026/BGG');
    await page.waitForSelector('.main');

    // Open filter drawer to reveal helper text
    const filterBtn = page.locator('button:has-text("Filters")');
    if (await filterBtn.isVisible()) {
      await filterBtn.click();
    }

    const hintFontSize = await page.evaluate(() => {
      const el = document.querySelector('.filter-hint-text, small.text-muted.d-block');
      return el ? parseFloat(getComputedStyle(el).fontSize) : 0;
    });

    // With 14px root font size and 0.875rem mobile token, font size should be >= 12px
    expect(hintFontSize).toBeGreaterThanOrEqual(12);

    const soldOutFontSize = await page.evaluate(() => {
      const el = document.querySelector('.eventGroup small.text-xs');
      return el ? parseFloat(getComputedStyle(el).fontSize) : 0;
    });

    if (soldOutFontSize > 0) {
      expect(soldOutFontSize).toBeGreaterThanOrEqual(12);
    }
  });

  test('Desktop: Compact data-dense layout is preserved', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/cat/2026/BGG');
    await page.waitForSelector('.main');

    const filterBtn = page.locator('button:has-text("Filters")');
    if (await filterBtn.isVisible()) {
      await filterBtn.click();
    }

    const desktopFontSize = await page.evaluate(() => {
      const el = document.querySelector('.filter-hint-text, small.text-muted.d-block');
      return el ? parseFloat(getComputedStyle(el).fontSize) : 0;
    });

    // Desktop font size should remain compact (approx 10.5px at 14px base font size for 0.75rem)
    expect(desktopFontSize).toBeLessThan(12);
  });

  test('Mobile: Filter drawer closes when clicking outside header', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/cat/2026/BGG');
    await page.waitForSelector('.main');

    const filterBtn = page.locator('button:has-text("Filters")');
    await filterBtn.click();

    const drawer = page.locator('.filter-drawer-wrapper');
    await expect(drawer).toHaveClass(/open/);

    // Tap/click outside the header/component area on the main body
    await page.mouse.click(200, 500);

    await expect(drawer).not.toHaveClass(/open/);
  });

  test('Mobile: Main navbar menu closes when clicking outside or navigating', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/cat/2026/BGG');
    await page.waitForSelector('.main');

    const navTogglerBtn = page.locator('button.navbar-toggler');
    await navTogglerBtn.click();

    const navCollapse = page.locator('#navToggler');
    await expect(navCollapse).toHaveClass(/show/);

    // Tap/click outside the navigation bar
    await page.mouse.click(200, 500);
    await expect(navCollapse).not.toHaveClass(/show/);
  });
});
