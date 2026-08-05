import { expect, Page, test } from '@playwright/test';

const username = process.env.LOGIN_USERNAME;
const password = process.env.LOGIN_PASSWORD;
const metricName = process.env.BASIC_ALERT_E2E_METRIC;
const channelName = process.env.BASIC_ALERT_E2E_CHANNEL;
const apiBaseURL =
	process.env.SIGINSIGHT_E2E_API_BASE_URL || 'http://localhost:8080';

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

	test.skip(
		!username || !password || !metricName || !channelName,
		'requires credentials plus BASIC_ALERT_E2E_METRIC and BASIC_ALERT_E2E_CHANNEL',
	);

	test('saves a v3 rule through the source backend and cleans it up', async ({
		page,
		request,
	}) => {
		let ruleID = '';
		let accessToken = '';
		try {
			await login(page);
			accessToken = await page.evaluate(
				() => localStorage.getItem('AUTH_TOKEN') || '',
			);
			expect(accessToken).not.toBe('');

			await page.goto('/alerts/new?alertType=METRIC_BASED_ALERT');
			await page.getByLabel('Alert name').fill('browser v3 alert verification');
			await page.getByLabel('Metric name').fill(metricName as string);

			const metricType = page
				.locator('.lite-query-control')
				.filter({ hasText: 'Metric type' });
			await metricType.locator('.ant-select-selector').click();
			await page
				.locator('.ant-select-dropdown:not(.ant-select-dropdown-hidden)')
				.getByText('Sum', { exact: true })
				.click();

			await page.getByLabel('critical threshold').fill('0');
			await page
				.getByRole('combobox', { name: 'Notification channels for critical' })
				.click();
			await page
				.locator('.ant-select-dropdown:not(.ant-select-dropdown-hidden)')
				.getByText(channelName as string, { exact: true })
				.click();

			const saveResponse = page.waitForResponse(
				(response) =>
					response.url().endsWith('/api/v5/rules') &&
					response.request().method() === 'POST',
			);
			await page.getByRole('button', { name: 'Save rule' }).click();
			const response = await saveResponse;
			expect(response.ok(), await response.text()).toBe(true);
			const body = (await response.json()) as {
				data?: { id?: string; schemaVersion?: string };
			};
			ruleID = body.data?.id || '';
			expect(ruleID).not.toBe('');
			expect(body.data?.schemaVersion).toBe('v3alpha1');
			await expect(page).toHaveURL(/\/alerts/, { timeout: 20_000 });
		} finally {
			if (ruleID && accessToken) {
				const deleteResponse = await request.delete(
					`${apiBaseURL}/api/v5/rules/${ruleID}`,
					{
						headers: { Authorization: `Bearer ${accessToken}` },
					},
				);
				expect(deleteResponse.ok(), await deleteResponse.text()).toBe(true);
			}
		}
	});
});
