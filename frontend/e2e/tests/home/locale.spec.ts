import { expect, test } from '@playwright/test';

const username = process.env.LOGIN_USERNAME;
const password = process.env.LOGIN_PASSWORD;

test.describe('home locale startup', () => {
	test.skip(
		!username || !password,
		'requires LOGIN_USERNAME and LOGIN_PASSWORD',
	);

	test('normalizes a POSIX locale before uPlot initializes', async ({
		page,
	}) => {
		await page.addInitScript(() => {
			Object.defineProperty(window.navigator, 'language', {
				configurable: true,
				value: 'en-US@posix',
			});
		});

		await page.goto('/home');
		await page.waitForURL(/\/login$/, { timeout: 15_000 });
		await page.getByTestId('email').fill(username as string);
		await page.getByTestId('initiate_login').click();
		await page.getByTestId('password').waitFor({
			state: 'visible',
			timeout: 15_000,
		});
		await page.getByTestId('password').fill(password as string);
		await page.getByTestId('password_authn_submit').click();

		await expect(page).toHaveURL(/\/home/, { timeout: 20_000 });
		await expect(page.getByText('Hello there, Welcome to your')).toBeVisible();
		await expect(page.getByText('Something went wrong')).not.toBeVisible();
		expect(await page.evaluate(() => window.navigator.language)).toBe('en-US');
	});
});
