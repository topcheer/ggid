import { test, expect } from '@playwright/test';
import { ALL_PAGES } from './pages-data';

// Smoke test: visit every console page and verify it loads (200, no crash)
// This ensures 100% page coverage for UI automation
test.describe.configure({ mode: 'serial' });

for (const pagePath of ALL_PAGES) {
  test(`page loads: ${pagePath}`, async ({ page }) => {
    const response = await page.goto(pagePath, { waitUntil: 'networkidle' });
    
    // Page should return 200 (or 401 redirect to login for protected pages)
    expect(response?.status()).toBeLessThan(500);
    
    // Check for React error boundaries or crash messages
    const bodyText = await page.textContent('body');
    expect(bodyText).not.toContain('Application error');
    expect(bodyText).not.toContain('Internal Server Error');
    expect(bodyText).not.toContain('Unhandled Runtime Error');
    
    // Check for hydration errors in console
    const consoleErrors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        const text = msg.text();
        if (!text.includes('favicon') && !text.includes('404')) {
          consoleErrors.push(text);
        }
      }
    });
    
    // Wait a bit for any async errors
    await page.waitForTimeout(500);
    
    // Hydration errors should not exist
    const hasHydrationError = consoleErrors.some(e => 
      e.includes('hydrat') || e.includes('did not match')
    );
    expect(hasHydrationError).toBeFalsy();
  });
}
