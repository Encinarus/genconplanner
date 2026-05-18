import { test, expect } from '@playwright/test';

test.describe('Party Hub E2E', () => {
  test.beforeEach(async ({ page }) => {
    // Mock user profile to simulate an authenticated session
    await page.addInitScript(() => {
      (window as any).serverSideUser = {
        email: 'leader@example.com',
        displayName: 'Party Leader',
        genconName: 'LeaderGencon',
        genconId: '12345',
        genconEmail: 'leader@gencon.com'
      };
    });

    // Mock API responses for parties
    await page.route('/api/v1/user', async route => {
      await route.fulfill({ json: { email: 'leader@example.com', displayName: 'Party Leader' } });
    });

    await page.route('/api/v1/user/parties', async route => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          json: {
            id: 101,
            name: 'Playwright Party',
            year: 2026,
            leaderEmail: 'leader@example.com',
            shortCode: 'PLWRT',
            inviteLink: 'http://localhost:4200/party/PLWRT',
            members: [{ email: 'leader@example.com', displayName: 'Party Leader' }]
          }
        });
      } else {
        await route.fulfill({
          status: 200,
          json: []
        });
      }
    });

    await page.route('/api/v1/party/*', async route => {
      if (route.request().url().includes('/interests')) {
        await route.fallback();
        return;
      }
      await route.fulfill({
        status: 200,
        json: {
          id: 101,
          name: 'Playwright Party',
          year: 2026,
          leaderEmail: 'leader@example.com',
          shortCode: 'PLWRT',
          inviteLink: 'http://localhost:4200/party/PLWRT',
          members: [{ email: 'leader@example.com', displayName: 'Party Leader' }]
        }
      });
    });

    await page.route('/api/v1/party/101/interests?year=2026', async route => {
      await route.fulfill({ status: 200, json: [] });
    });
  });

  test('should navigate to Party Hub, create a party, and view details', async ({ page }) => {
    await page.goto('/user');

    // Check that user profile is visible
    await expect(page.locator('h1', { hasText: 'Party Leader' })).toBeVisible();

    // Create a new party
    const partyInput = page.locator('input#partyName');
    await partyInput.fill('Playwright Party');
    await page.locator('button', { hasText: 'Create Party' }).click();

    // Verify party appears in the list
    const partyCard = page.locator('.party-card', { hasText: 'Playwright Party' });
    await expect(partyCard).toBeVisible();

    // Click to view party details
    await partyCard.locator('a', { hasText: 'View Details' }).click();
    await expect(page).toHaveURL(/.*\/party\/2026\/events/);
    await expect(page.locator('h1', { hasText: 'Playwright Party' })).toBeVisible();

    // Switch to Members tab
    await page.locator('button[title="Party Members"]').click();
    await expect(page).toHaveURL(/.*\/party\/2026\/members/);
    await expect(page.locator('.contact-card', { hasText: 'Party Leader' })).toBeVisible();
  });
});
