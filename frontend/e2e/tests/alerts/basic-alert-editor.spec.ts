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

async function selectAlertDataSource(
	page: Page,
	dataSource: 'Metrics' | 'Logs' | 'Traces' | 'Exceptions',
): Promise<void> {
	await page;
	await page
		.locator('.basic-alert-editor__query-tabs')
		.getByRole('button', { name: dataSource, exact: true })
		.click();
}

const traceQueryWithLogFilter = {
	id: 'e2e-alert-signal-switch',
	queryType: 'builder',
	builder: {
		queryData: [
			{
				queryName: 'A',
				dataSource: 'traces',
				aggregateOperator: 'count',
				aggregateAttribute: { id: '', key: '', dataType: '', type: '' },
				aggregations: [{ expression: 'count()' }],
				functions: [],
				filters: { items: [], op: 'AND' },
				filter: { expression: "severity_text != 'INFO'" },
				expression: 'A',
				disabled: false,
				stepInterval: 60,
				having: [],
				limit: null,
				orderBy: [],
				groupBy: [],
				legend: '',
			},
		],
		queryFormulas: [],
	},
	clickhouse_sql: [],
	unit: '',
};

function alertQuerySignal(page: Page): Promise<string> {
	return page.evaluate(() => {
		const compositeQuery = new URL(window.location.href).searchParams.get(
			'compositeQuery',
		);
		if (!compositeQuery) {
			return '';
		}
		return JSON.parse(decodeURIComponent(compositeQuery)).builder.queryData[0]
			.dataSource;
	});
}

test.describe('basic alert editor', () => {
	test('switching the alert signal rewrites a stale composite query', async ({
		page,
	}) => {
		test.skip(
			!username || !password,
			'requires LOGIN_USERNAME and LOGIN_PASSWORD',
		);
		await login(page);
		await page.goto(
			`/alerts/new?compositeQuery=${encodeURIComponent(
				JSON.stringify(traceQueryWithLogFilter),
			)}`,
		);
		await expect(
			page.getByRole('heading', { name: 'New alert rule' }),
		).toBeVisible({ timeout: 20_000 });

		await selectAlertDataSource(page, 'Logs');
		await expect.poll(() => alertQuerySignal(page)).toBe('logs');
	});

	test('renders the v3 editor without legacy scheduling controls', async ({
		page,
	}) => {
		test.skip(
			!username || !password,
			'requires LOGIN_USERNAME and LOGIN_PASSWORD',
		);
		await login(page);
		await page.goto('/alerts/new?alertType=METRIC_BASED_ALERT');

		await expect(
			page.getByRole('heading', { name: 'New alert rule' }),
		).toBeVisible({ timeout: 20_000 });
		await expect(page.getByLabel('Alert name')).toBeVisible();
		await expect(page.getByTestId('lite-query-builder')).toBeVisible();
		const queryPreview = page.getByTestId('alert-query-preview');
		await expect(queryPreview).toBeVisible();
		await expect(queryPreview.getByText('Chart Preview')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Run preview' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Test rule' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Save rule' })).toBeVisible();
		const notificationSettings = page
			.locator('section')
			.filter({ hasText: 'Notification settings' });
		await expect(
			notificationSettings.getByRole('combobox', {
				name: 'Notification channel',
			}),
		).toBeVisible();
		await expect(
			page
				.locator('section')
				.filter({ hasText: 'Condition and evaluation' })
				.getByRole('combobox', {
					name: 'Notification channel',
				}),
		).toHaveCount(0);
		await expect(
			page.getByText(/custom schedule|rrule|starting at/i),
		).toHaveCount(0);
		await expect(page.getByText(/switch to classic experience/i)).toHaveCount(0);

		await page.getByLabel('Alert name').fill('preserved alert draft');
		await page.getByLabel('Alert label key').fill('team');
		await page.getByLabel('Alert label value').fill('platform');
		await page.getByRole('button', { name: 'Add label' }).click();
		await expect(page.getByText('team: platform')).toBeVisible();
		await page.getByLabel('Metric name').fill('preserved.metric');
		await selectAlertDataSource(page, 'Logs');
		await expect(page.getByLabel('Alert name')).toHaveValue(
			'preserved alert draft',
		);
		await selectAlertDataSource(page, 'Metrics');
		await expect(page.getByLabel('Metric name')).toHaveValue('preserved.metric');

		await page.getByRole('button', { name: 'Run preview' }).click();
		await expect(queryPreview).not.toContainText(
			'Run preview to view the current query',
			{
				timeout: 20_000,
			},
		);
	});

	test('saves a v3 rule through the source backend and cleans it up', async ({
		page,
		request,
	}) => {
		test.skip(
			!username || !password || !metricName || !channelName,
			'requires credentials plus BASIC_ALERT_E2E_METRIC and BASIC_ALERT_E2E_CHANNEL',
		);
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
			await page.getByRole('button', { name: 'Add severity' }).click();
			await page.getByLabel('warning threshold').fill('0');
			await page.getByRole('combobox', { name: 'Notification channel' }).click();
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
				data?: {
					id?: string;
					schemaVersion?: string;
					condition?: {
						numeric?: { thresholds?: { channels?: string[] }[] };
					};
				};
			};
			ruleID = body.data?.id || '';
			expect(ruleID).not.toBe('');
			expect(body.data?.schemaVersion).toBe('v3alpha1');
			const savedThresholds = body.data?.condition?.numeric?.thresholds || [];
			expect(savedThresholds).toHaveLength(2);
			expect(
				savedThresholds.every((threshold) => threshold.channels?.length === 1),
			).toBe(true);
			expect([
				...new Set(savedThresholds.map((threshold) => threshold.channels?.[0])),
			]).toHaveLength(1);
			await expect(page).toHaveURL(/\/alerts/, { timeout: 20_000 });

			await page.goto(`/alerts/overview?ruleId=${ruleID}`);
			await expect(page.getByText('Overview', { exact: true })).toBeVisible({
				timeout: 20_000,
			});
			await expect(page.getByText('History', { exact: true })).toBeVisible();

			const historyResponses = [
				'history/stats',
				'history/timeline',
				'history/top_contributors',
				'history/overall_status',
			].map((path) =>
				page.waitForResponse(
					(response) =>
						response.url().includes(`/api/v5/rules/${ruleID}/${path}`) &&
						response.request().method() === 'POST',
					{ timeout: 20_000 },
				),
			);
			await page.getByText('History', { exact: true }).click();
			const responses = await Promise.all(historyResponses);
			responses.forEach((historyResponse) =>
				expect(historyResponse.ok(), historyResponse.url()).toBe(true),
			);
			await expect(page).toHaveURL(/\/alerts\/history\?/, {
				timeout: 20_000,
			});
			await expect(page.getByText('TOTAL TRIGGERED')).toBeVisible();
			await expect(page.getByText('Timeline', { exact: true })).toBeVisible();
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
