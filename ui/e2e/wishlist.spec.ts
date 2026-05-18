import { test, expect } from '@playwright/test';

test.describe('Wishlist Prioritization Engine E2E', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      (window as any).serverSideUser = { email: 'leader@example.com', displayName: 'Party Leader' };
    });

    await page.route('/api/v1/user', async route => {
      await route.fulfill({ json: { email: 'leader@example.com', displayName: 'Party Leader' } });
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
            status: 'Scheduled',
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
    await page.goto('/wishlist/2026');

    // Verify wishlist item appears with score and reasoning
    const itemCard = page.locator('.wishlist-item', { hasText: 'Catan Championship' });
    await expect(itemCard).toBeVisible();
    await expect(itemCard.locator('.score-badge')).toHaveText('Score: 70');
    await expect(itemCard.locator('li', { hasText: 'Must Have interest (+50 pts)' })).toBeVisible();

    // Verify existing constraint is displayed
    const constraintRow = page.locator('.constraint-item', { hasText: 'Every Day' });
    await expect(constraintRow).toBeVisible();
    await expect(constraintRow).toContainText('12:00 PM - 01:00 PM (Min: 30 mins)');

    // Add a new flexible break constraint
    await page.locator('select.day-select').selectOption({ label: 'Friday' });
    await page.locator('input.start-time').fill('17:00');
    await page.locator('input.end-time').fill('18:30');
    await page.locator('input.min-duration').fill('45');
    await page.locator('button', { hasText: 'Add Break' }).click();

    // Verify the new constraint appears
    await expect(page.locator('.constraint-item', { hasText: 'Friday' })).toBeVisible();
    await expect(page.locator('.constraint-item', { hasText: 'Friday' })).toContainText('05:00 PM - 06:30 PM (Min: 45 mins)');

    // Save constraints
    await page.locator('button', { hasText: 'Save Constraints' }).click();
    await expect(page.locator('.alert-success', { hasText: 'Constraints saved successfully!' })).toBeVisible();
  });
});
