import { rest } from 'msw';

import commonEnTranslation from '../../public/locales/en/common.json';
import enTranslation from '../../public/locales/en/translation.json';
import { allAlertChannels } from './__mockdata__/alerts';
import { explorerView } from './__mockdata__/explorer_views';
import { membersResponse } from './__mockdata__/members';
import { queryRangeSuccessResponse } from './__mockdata__/query_range';
import { serviceSuccessResponse } from './__mockdata__/services';
import { topLevelOperationSuccessResponse } from './__mockdata__/top_level_operations';

export const handlers = [
	rest.post('http://localhost/api/v5/query_range', (req, res, ctx) =>
		res(ctx.status(200), ctx.json(queryRangeSuccessResponse)),
	),

	rest.post('http://localhost/api/v5/services', (req, res, ctx) =>
		res(
			ctx.status(200),
			ctx.json({ status: 'success', data: serviceSuccessResponse }),
		),
	),

	rest.post(
		'http://localhost/api/v5/service/top_level_operations',
		(req, res, ctx) =>
			res(ctx.status(200), ctx.json(topLevelOperationSuccessResponse)),
	),

	rest.get('http://localhost/api/v5/users/me', (req, res, ctx) =>
		res(ctx.status(200), ctx.json({ status: '200', data: membersResponse })),
	),
	rest.get(
		'http://localhost/api/v5/autocomplete/attribute_keys',
		(req, res, ctx) => {
			const metricName = req.url.searchParams.get('metricName');
			const match = req.url.searchParams.get('match');

			if (metricName === 'signoz_calls_total' && match === 'resource_') {
				return res(
					ctx.status(200),
					ctx.json({ status: 'success', data: ['resource_signoz_collector_id'] }),
				);
			}

			return res(ctx.status(500));
		},
	),

	rest.get(
		'http://localhost/api/v5/autocomplete/attribute_values',
		(req, res, ctx) => {
			// ?metricName=signoz_calls_total&tagKey=resource_signoz_collector_id
			const metricName = req.url.searchParams.get('metricName');
			const tagKey = req.url.searchParams.get('tagKey');

			const attributeKey = req.url.searchParams.get('attributeKey');

			if (attributeKey === 'service.name') {
				return res(
					ctx.status(200),
					ctx.json({
						status: 'success',
						data: {
							stringAttributeValues: [
								'customer',
								'demo-app',
								'driver',
								'frontend',
								'mysql',
								'redis',
								'route',
								'go-grpc-otel-server',
								'test',
							],
							numberAttributeValues: null,
							boolAttributeValues: null,
						},
					}),
				);
			}

			if (attributeKey === 'name') {
				return res(
					ctx.status(200),
					ctx.json({
						status: 'success',
						data: {
							stringAttributeValues: [
								'HTTP GET',
								'HTTP GET /customer',
								'HTTP GET /dispatch',
								'HTTP GET /route',
							],
							numberAttributeValues: null,
							boolAttributeValues: null,
						},
					}),
				);
			}

			if (attributeKey === 'http_method') {
				return res(
					ctx.status(200),
					ctx.json({
						status: 'success',
						data: {
							stringAttributeValues: ['GET', 'POST'],
							numberAttributeValues: null,
							boolAttributeValues: null,
						},
					}),
				);
			}

			if (attributeKey === 'http.route') {
				return res(
					ctx.status(200),
					ctx.json({
						status: 'success',
						data: {
							stringAttributeValues: ['/health', '/v1/orders'],
							numberAttributeValues: null,
							boolAttributeValues: null,
						},
					}),
				);
			}

			if (
				metricName === 'signoz_calls_total' &&
				tagKey === 'resource_signoz_collector_id'
			) {
				return res(
					ctx.status(200),
					ctx.json({
						status: 'success',
						data: [
							'f38916c2-daf2-4424-bd3e-4907a7e537b6',
							'6d4af7f0-4884-4a37-abd4-6bdbee29fa04',
							'523c44b9-5fe1-46f7-9163-4d2c57ece09b',
							'aa52e8e8-6f88-4056-8fbd-b377394d022c',
							'4d515ba2-065d-4856-b2d8-ddb957c44ddb',
							'fd47a544-1410-4c76-a554-90ef6464da02',
							'bb455f71-3fe1-4761-bbf5-efe2faee18a6',
							'48563680-314e-4117-8a6d-1f0389c95e04',
							'6e866423-7704-4d72-be8b-4695bc36f145',
							'e4886c76-93f5-430f-9076-eef85524312f',
						],
					}),
				);
			}

			return res(ctx.status(500));
		},
	),
	rest.post('http://localhost/api/v5/invite', (_, res, ctx) =>
		res(
			ctx.status(200),
			ctx.json({
				status: 'success',
				data: 'invite sent successfully',
			}),
		),
	),
	rest.put('http://localhost/api/v5/user/:id', (_, res, ctx) =>
		res(
			ctx.status(200),
			ctx.json({
				data: 'user updated successfully',
			}),
		),
	),
	rest.post('http://localhost/api/v5/changePassword', (_, res, ctx) =>
		res(
			ctx.status(403),
			ctx.json({
				status: 'error',
				errorType: 'forbidden',
				error: 'invalid credentials',
			}),
		),
	),

	rest.get(
		'http://localhost/api/v5/autocomplete/aggregate_attributes',
		(req, res, ctx) =>
			res(
				ctx.status(200),
				ctx.json({
					status: 'success',
					data: { attributeKeys: null },
				}),
			),
	),

	rest.get('http://localhost/api/v5/explorer/views', (req, res, ctx) =>
		res(ctx.status(200), ctx.json(explorerView)),
	),

	rest.post('http://localhost/api/v5/explorer/views', (req, res, ctx) =>
		res(
			ctx.status(200),
			ctx.json({
				status: 'success',
				data: '7731ece1-3fa3-4ed4-8b1c-58b4c28723b2',
			}),
		),
	),

	rest.post('http://localhost/api/v5/event', (req, res, ctx) =>
		res(
			ctx.status(200),
			ctx.json({
				statusCode: 200,
				error: null,
				payload: 'Event Processed Successfully',
			}),
		),
	),

	rest.get('http://localhost/api/v5/channels', (_, res, ctx) =>
		res(ctx.status(200), ctx.json({ data: allAlertChannels, status: 'success' })),
	),
	rest.delete('http://localhost/api/v5/channels/:id', (_, res, ctx) =>
		res(
			ctx.status(200),
			ctx.json({
				status: 'success',
				data: 'notification channel successfully deleted',
			}),
		),
	),
	rest.get('http://localhost/locales/en/translation.json', (_, res, ctx) =>
		res(ctx.status(200), ctx.json(enTranslation)),
	),
	rest.get('http://localhost/locales/en/common.json', (_, res, ctx) =>
		res(ctx.status(200), ctx.json(commonEnTranslation)),
	),
	rest.get('http://localhost/locales/en-US/translation.json', (_, res, ctx) =>
		res(ctx.status(200), ctx.json(enTranslation)),
	),
	rest.get('http://localhost/locales/en-US/common.json', (_, res, ctx) =>
		res(ctx.status(200), ctx.json(commonEnTranslation)),
	),
];
