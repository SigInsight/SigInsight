import { expect, Page, test } from '@playwright/test';

const username = process.env.LOGIN_USERNAME;
const password = process.env.LOGIN_PASSWORD;

async function login(page: Page): Promise<void> {
	await page.goto('/login');
	await page.getByTestId('email').fill(username as string);
	await page.getByTestId('initiate_login').click();
	await page.getByTestId('password').waitFor({ state: 'visible' });
	await page.getByTestId('password').fill(password as string);
	await page.getByTestId('password_authn_submit').click();
	await expect(page).toHaveURL(/\/home/, { timeout: 20_000 });
}

test.describe('basic alert editor', () => {
	test.skip(
		!username || !password,
		'requires LOGIN_USERNAME and LOGIN_PASSWORD',
	);

	test('renders the v3 editor without legacy scheduling controls', async ({
		page,
	}) => {
		await login(page);
		await page.goto('/alerts/new?alertType=METRIC_BASED_ALERT');

		await expect(
			page.getByRole('heading', { name: 'New alert rule' }),
		).toBeVisible({ timeout: 20_000 });
		await expect(page.getByLabel('Alert name')).toBeVisible();
		await expect(page.getByTestId('lite-query-builder')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Run preview' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Test rule' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Save rule' })).toBeVisible();
		await expect(
			page.getByText(/custom schedule|rrule|starting at/i),
		).toHaveCount(0);
		await expect(page.getByText(/switch to classic experience/i)).toHaveCount(0);
	});
});
