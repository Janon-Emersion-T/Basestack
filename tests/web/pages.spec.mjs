import { test, expect } from '@playwright/test';

test('navigate between pages, refresh, back and handle missing routes', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Your next idea starts here.');
  await page.getByRole('navigation').getByRole('link', { name: 'About us', exact: true }).click();
  await expect(page).toHaveURL(/\?page=about$/);
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('About us');
  await expect(page).toHaveTitle('About us | smoke-app');
  await expect(page.getByRole('navigation').getByRole('link', { name: 'About us', exact: true })).toHaveAttribute('aria-current', 'page');
  await page.reload();
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('About us');
  await page.goBack();
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Your next idea starts here.');
  await page.goto('/?page=missing');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Page not found');
  await page.getByRole('link', { name: 'Return home' }).click();
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Your next idea starts here.');
  expect(errors).toEqual([]);
});

test('all alternate layouts render without horizontal overflow', async ({ page }) => {
  await page.goto('/?page=about');
  for (const variant of ['minimal', 'split', 'list', 'compact', 'centered']) {
    await expect(page.locator(`.variant-${variant}`).first()).toBeVisible();
  }
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.getByRole('navigation').getByRole('link', { name: 'contact', exact: true }).click();
  await expect(page).toHaveURL(/\?page=about#contact$/);
  await expect(page.locator('#contact')).toBeInViewport();
  await expect(page.getByRole('link', { name: 'Email us' })).toHaveAttribute('href', 'mailto:hello@example.com');
});

test('empty page and keyboard skip link remain usable', async ({ page }) => {
  await page.goto('/?page=empty');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Your canvas is ready.');
  await page.getByRole('link', { name: 'Return home' }).click();
  await page.keyboard.press('Tab');
  await expect(page.getByRole('link', { name: 'Skip to content' })).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('main')).toBeFocused();
});
