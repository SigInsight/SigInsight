import { expect, Page, test } from '@playwright/test';

const username = process.env.LOGIN_USERNAME;
const password = process.env.LOGIN_PASSWORD;

type QueryRangeCall = {
	status: number;
	payload: Record<string, unknown>;
	body: string;
};

function observeClientErrors(page: Page): string[] {
	const errors: string[] = [];
	page.on('pageerror', (error) => {
		errors.push(error.stack || error.message);
	});
	page.on('console', (message) => {
		if (message.type() === 'error') {
			errors.push(message.text());
		}
	});
	return errors;
}

async function login(page: Page): Promise<void> {
	await page.goto('/login');
	await page.getByTestId('email').fill(username as string);
	await page.getByTestId('initiate_login').click();
	await page.getByTestId('password').waitFor({ state: 'visible' });
	await page.getByTestId('password').fill(password as string);
	await page.getByTestId('password_authn_submit').click();
	await expect(page).toHaveURL(/\/home/, { timeout: 20_000 });
}

function observeQueryRange(page: Page): QueryRangeCall[] {
	const calls: QueryRangeCall[] = [];
	page.on('response', async (response) => {
		if (!response.url().includes('/api/v5/query_range')) {
			return;
		}
		const request = response.request();
		let payload: Record<string, unknown> = {};
		try {
			payload = JSON.parse(request.postData() || '{}') as Record<string, unknown>;
		} catch {
			// A non-JSON request is still retained as an observable test failure.
		}
		let body = '';
		try {
			body = await response.text();
		} catch {
			// Navigation may release the browser resource before the optional body
			// is read. Status and request payload remain sufficient for assertions.
		}
		calls.push({
			status: response.status(),
			payload,
			body,
		});
	});
	return calls;
}

async function expectSuccessfulQueryRange(
	page: Page,
	calls: QueryRangeCall[],
): Promise<void> {
	await expect.poll(() => calls.length, { timeout: 30_000 }).toBeGreaterThan(0);
	const failed = calls.filter((call) => call.status < 200 || call.status >= 300);
	expect(failed, JSON.stringify(failed, null, 2)).toEqual([]);
	await expect(
		page.getByText(
			/invalid lightweight query|invalid_input|unsupported lightweight query capability/i,
		),
	).toHaveCount(0);
}

async function runQueryAndExpectPayload(
	page: Page,
	calls: QueryRangeCall[],
	payloadFragment: string,
): Promise<void> {
	const firstNewCall = calls.length;
	await expect(page.getByRole('button', { name: 'Run Query' })).toBeVisible({
		timeout: 30_000,
	});
	await page.getByRole('button', { name: 'Run Query' }).click();
	await expect
		.poll(
			() =>
				calls
					.slice(firstNewCall)
					.filter((call) => JSON.stringify(call.payload).includes(payloadFragment)),
			{ timeout: 30_000 },
		)
		.not.toHaveLength(0);

	const matchingCalls = calls
		.slice(firstNewCall)
		.filter((call) => JSON.stringify(call.payload).includes(payloadFragment));
	expect(
		matchingCalls.filter((call) => call.status < 200 || call.status >= 300),
		JSON.stringify(matchingCalls, null, 2),
	).toEqual([]);
	await expect(
		page.getByText(
			/invalid lightweight query|invalid_input|unsupported lightweight query capability/i,
		),
	).toHaveCount(0);
}

async function replaceUnsupportedQuery(page: Page): Promise<void> {
	const replaceButton = page.getByRole('button', { name: 'Replace query' });
	if (await replaceButton.count()) {
		await replaceButton.click();
	}
}

async function selectLiteControl(
	page: Page,
	label: string,
	option: string,
): Promise<void> {
	const control = page
		.locator('.lite-query-control')
		.filter({ has: page.getByText(label, { exact: true }) });
	await control.locator('.ant-select-selector').click();
	await page
		.locator('.ant-select-dropdown:not(.ant-select-dropdown-hidden)')
		.getByText(option, { exact: true })
		.click();
}

test.describe('lightweight query explorer', () => {
	test.skip(
		!username || !password,
		'requires LOGIN_USERNAME and LOGIN_PASSWORD',
	);

	test.beforeEach(async ({ page }) => {
		await login(page);
	});

	test('loads logs and requests the next raw page while scrolling', async ({
		page,
	}) => {
		const calls = observeQueryRange(page);
		await page.goto('/logs/logs-explorer?relativeTime=1d');
		await expectSuccessfulQueryRange(page, calls);

		const scroller = page.locator('[data-test-id="virtuoso-scroller"]');
		await expect(scroller).toBeVisible({ timeout: 20_000 });
		await scroller.evaluate((element) => {
			element.scrollTop = element.scrollHeight;
			element.dispatchEvent(new Event('scroll', { bubbles: true }));
		});

		await expect
			.poll(
				() =>
					calls.some((call) =>
						JSON.stringify(call.payload).includes('"offset":100'),
					),
				{ timeout: 30_000 },
			)
			.toBe(true);
		await expectSuccessfulQueryRange(page, calls);
	});

	test('runs a filtered logs query through the lightweight protocol', async ({
		page,
	}) => {
		const calls = observeQueryRange(page);
		await page.goto('/logs/logs-explorer');
		await expectSuccessfulQueryRange(page, calls);
		await replaceUnsupportedQuery(page);
		await expect(page.getByTestId('lite-query-builder')).toBeVisible({
			timeout: 20_000,
		});

		await page
			.getByRole('textbox', { name: 'Filter expression for A' })
			.fill("severity_text = 'ERROR'");
		await runQueryAndExpectPayload(page, calls, 'severity_text');
	});

	test('keeps query composition actions inside their supported panel boundary', async ({
		page,
	}) => {
		await page.goto('/logs/logs-explorer');
		await replaceUnsupportedQuery(page);
		await expect(page.getByTestId('lite-query-builder')).toBeVisible({
			timeout: 20_000,
		});
		await expect(page.getByRole('button', { name: 'Add query' })).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Add formula' })).toHaveCount(
			0,
		);
		await expect(
			page.getByRole('button', { name: 'Duplicate query A' }),
		).toHaveCount(0);

		const quickFilterAnnouncement = page.getByRole('button', { name: 'Okay' });
		if (await quickFilterAnnouncement.count()) {
			await quickFilterAnnouncement.click();
		}
		await page.getByRole('button', { name: 'Time Series' }).click();
		await expect(page.getByRole('button', { name: 'Add query' })).toBeVisible();
		await page.getByRole('button', { name: 'Duplicate query A' }).click();
		await page.getByRole('button', { name: 'Add query' }).click();
		await page.getByRole('button', { name: 'Add formula' }).click();

		await expect(page.getByTestId('lite-query-builder')).toBeVisible();
		await expect(
			page.getByText(
				'This saved query uses capabilities that are not supported by the lightweight query engine.',
			),
		).toHaveCount(0);
	});

	test('loads the trace explorer without a lightweight validation error', async ({
		page,
	}) => {
		const calls = observeQueryRange(page);
		await page.goto('/traces-explorer');
		await expectSuccessfulQueryRange(page, calls);
	});

	test('runs a typed trace numeric filter through the shared expression editor', async ({
		page,
	}) => {
		const calls = observeQueryRange(page);
		await page.goto('/traces-explorer');
		await expectSuccessfulQueryRange(page, calls);
		await replaceUnsupportedQuery(page);
		await expect(page.getByTestId('lite-query-builder')).toBeVisible({
			timeout: 20_000,
		});

		await page
			.getByRole('textbox', { name: 'Filter expression for A' })
			.fill('duration_nano > 0');
		await runQueryAndExpectPayload(page, calls, 'duration_nano');
	});

	test('renders the metric explorer query builder', async ({ page }) => {
		const calls = observeQueryRange(page);
		await page.goto('/metrics-explorer/explorer');
		await expect(page.getByTestId('lite-query-builder')).toBeVisible({
			timeout: 20_000,
		});
		await expect(page.getByLabel('Metric name')).toBeVisible();
		await page
			.getByLabel('Metric name')
			.fill('http.server.request.duration.bucket');
		await selectLiteControl(page, 'Metric type', 'Histogram');
		await selectLiteControl(page, 'Aggregate', 'p95');
		await runQueryAndExpectPayload(
			page,
			calls,
			'http.server.request.duration.bucket',
		);
	});

	test('opens an alert rule type with the lightweight query builder', async ({
		page,
	}) => {
		const clientErrors = observeClientErrors(page);
		await page.goto('/alerts/type-selection');
		await page.getByTestId('alert-type-card-METRIC_BASED_ALERT').click();
		try {
			await expect(page.getByTestId('lite-query-builder')).toBeVisible({
				timeout: 20_000,
			});
		} catch (error) {
			throw new Error(
				`${String(error)}\nClient errors:\n${clientErrors.join('\n')}`,
			);
		}
	});

	test('switches alert signals and keeps the Meter query builder usable', async ({
		page,
	}) => {
		observeQueryRange(page);
		await page.goto('/alerts/type-selection');
		await page.getByTestId('alert-type-card-METRIC_BASED_ALERT').click();
		await expect(page.getByTestId('lite-query-builder')).toBeVisible({
			timeout: 20_000,
		});

		await selectLiteControl(page, 'Source', 'Meter');
		const cancelButton = page.getByRole('button', { name: 'Cancel' });
		if (await cancelButton.count()) {
			await cancelButton.click();
		}
		await page.getByLabel('Metric name').fill('signoz.meter.log.count');
		await page.getByLabel('Metric name').press('Tab');
		await expect(page.getByLabel('Metric name')).toHaveValue(
			'signoz.meter.log.count',
		);
		await expect(
			page.getByText(
				/invalid lightweight query|invalid_input|unsupported lightweight query capability/i,
			),
		).toHaveCount(0);

		await page.getByRole('button', { name: 'Logs' }).click();
		await expect(page.getByLabel('Metric name')).toHaveCount(0);
		await expect(page.getByTestId('lite-query-builder')).toBeVisible();

		await page.getByRole('button', { name: 'Traces' }).click();
		await expect(page.getByLabel('Metric name')).toHaveCount(0);
		await expect(page.getByTestId('lite-query-builder')).toBeVisible();
	});
});
