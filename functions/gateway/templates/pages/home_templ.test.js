import { test, expect } from '@playwright/test';

test('filter events by Academic category', async ({ page }) => {
    // Navigate to test server
    await page.goto(process.env.TEST_SERVER_URL);

    // Click filter button
    await page.click('button:has-text("FILTER")');

    // Click Academic category
    await page.click('text=Academic & Career Development');

    // Click apply filters
    await page.click('button:has-text("Apply Filters")');

    // Verify filtered results
    await expect(page.locator('text=Academic Workshop')).toBeVisible();
    await expect(page.locator('text=Career Fair')).toBeVisible();
    await expect(page.locator('text=Test Event 1')).not.toBeVisible();
});
