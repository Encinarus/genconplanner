import { test, expect } from '@playwright/test';

test.describe('Wishlist Prioritization Engine E2E', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      (window as any).serverSideUser = { email: 'leader@example.com', displayName: 'Party Leader' };
    });

    await page.route('/api/v1/user', async route => {
      await route.fulfill({ json: { email: 'leader@example.com', displayName: 'Party Leader' } });
    });

    await page.route('/api/v1/user/starred/page/2026', async route => {
      await route.fulfill({
        status: 200,
        json: {
          email: 'leader@example.com',
          year: 2026,
          calendarEvents: [],
          individualEvents: [],
          metadata: { startDate: '2026-07-30', endDate: '2026-08-02' },
          starredClusters: [],
          starredEvents: []
        }
      });
    });

    // Mock wishlist items
    await page.route('/api/v1/user/wishlist/2026', async route => {
      await route.fulfill({
        status: 200,
        json: [
          {
            event: {
              eventId: 'BGM101',
              title: 'Catan Championship',
              shortDescription: 'Play Catan',
              categoryCode: 'BGM',
              startTime: '2026-07-30T10:00:00Z',
              endTime: '2026-07-30T12:00:00Z',
              genconUrl: 'http://gencon.com/BGM101',
              plannerUrl: '/legacy/event/BGM101',
              tier: 'must_have',
              groupTier: 'must_have',
              isOverride: false,
              location: 'ICC',
              roomName: 'Hall A',
              tableNumber: '1'
            },
            status: 'Primary',
            reasoning: ['Must Have interest (+50 pts)', 'Few tickets left (+20 pts)'],
            score: 70
          }
        ]
      });
    });

    // Mock wishlist constraints
    await page.route('/api/v1/user/wishlist/constraints', async route => {
      if (route.request().method() === 'POST') {
        await route.fulfill({ status: 200, json: { success: true } });
      } else {
        await route.fulfill({
          status: 200,
          json: [
            {
              dayOfWeek: -1,
              startHour: 12,
              startMinute: 0,
              endHour: 13,
              endMinute: 0,
              minDurationMinutes: 30
            }
          ]
        });
      }
    });

    await page.route('/api/v1/user/parties', async route => {
      await route.fulfill({ status: 200, json: [] });
    });
  });

  test('should view prioritized wishlist and update flexible break constraints', async ({ page }) => {
    await page.goto('/starred/2026/wishlist');

    // Verify wishlist item appears with rank and reasoning
    const itemCard = page.locator('.wishlist-item', { hasText: 'Catan Championship' });
    await expect(itemCard).toBeVisible();
    await expect(itemCard.locator('.wishlist-rank')).toHaveText('#1');
    await expect(itemCard.locator('.reasoning-badge', { hasText: 'Few tickets left (+20 pts)' })).toBeVisible();

    // Verify existing constraint is displayed
    const constraintRow = page.locator('.constraint-row').first();
    await expect(constraintRow).toBeVisible();
    await expect(constraintRow.locator('select').nth(0).locator('option:checked')).toHaveText('Every Day');
    await expect(constraintRow.locator('select').nth(1).locator('option:checked')).toHaveText('Noon');
    await expect(constraintRow.locator('input[type="number"]')).toHaveValue('30');

    // Add a new flexible break constraint
    await page.locator('button', { hasText: 'Add Time Block' }).click();
    const newConstraint = page.locator('.constraint-row').nth(1);
    await expect(newConstraint).toBeVisible();

    await newConstraint.locator('select').nth(0).selectOption({ label: 'Friday' });
    await newConstraint.locator('select').nth(1).selectOption({ label: '5 PM' });
    await newConstraint.locator('select').nth(2).selectOption({ label: ':00' });
    await newConstraint.locator('select').nth(3).selectOption({ label: '6 PM' });
    await newConstraint.locator('select').nth(4).selectOption({ label: ':30' });
    await newConstraint.locator('input[type="number"]').fill('45');

    // Verify the new constraint values
    await expect(newConstraint.locator('select').nth(0).locator('option:checked')).toHaveText('Friday');
    await expect(newConstraint.locator('input[type="number"]')).toHaveValue('45');

    // Wait for auto-save debounce (1500ms)
    await page.waitForTimeout(2000);
  });
});
