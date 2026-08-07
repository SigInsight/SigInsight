import { useState } from 'react';
import { QueryClient, QueryClientProvider } from 'react-query';
import { MemoryRouter } from 'react-router-dom';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryBuilderContext } from 'providers/QueryBuilder';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { QueryBuilderContextType } from 'types/common/queryBuilder';

import BasicAlertEditor from './BasicAlertEditor';
import { defaultQueryForAlertType } from './draft';

jest.mock('api/channels/getAll', () => jest.fn(async () => ({ data: [] })));

jest.mock('features/lite-query/LiteQueryBuilder', () => ({
	LiteQueryBuilder: (): JSX.Element => <div data-testid="lite-query-builder" />,
}));

jest.mock('./AlertQueryPreview', () => ({
	__esModule: true,
	default: ({
		query,
		runID,
	}: {
		query: Query | null;
		runID: number;
	}): JSX.Element => (
		<div data-testid="alert-query-preview-state">
			{query
				? `${query.builder.queryData[0].dataSource}:${query.builder.queryData[0].stepInterval}:${runID}`
				: 'none'}
		</div>
	),
}));

function AlertEditorHarness({
	onRedirect,
	onReset,
}: {
	onRedirect: jest.Mock;
	onReset: jest.Mock;
}): JSX.Element {
	const [currentQuery, setCurrentQuery] = useState<Query>(
		defaultQueryForAlertType(AlertTypes.METRICS_BASED_ALERT),
	);
	const resetQuery = (nextQuery?: Query): void => {
		onReset(nextQuery);
		if (nextQuery) {
			setCurrentQuery(nextQuery);
		}
	};
	const context = ({
		currentQuery,
		redirectWithQueryBuilderData: onRedirect,
		resetQuery,
	} as unknown) as QueryBuilderContextType;

	return (
		<QueryClientProvider client={new QueryClient()}>
			<MemoryRouter>
				<QueryBuilderContext.Provider value={context}>
					<BasicAlertEditor alertType={AlertTypes.METRICS_BASED_ALERT} />
				</QueryBuilderContext.Provider>
			</MemoryRouter>
		</QueryClientProvider>
	);
}

describe('BasicAlertEditor preview lifecycle', () => {
	it('runs only an explicit, interval-normalized query and clears it on signal changes', async () => {
		const onRedirect = jest.fn();
		const onReset = jest.fn();
		render(<AlertEditorHarness onRedirect={onRedirect} onReset={onReset} />);

		await waitFor(() =>
			expect(onReset).toHaveBeenCalledWith(
				expect.objectContaining({
					builder: expect.objectContaining({
						queryData: [expect.objectContaining({ stepInterval: 60 })],
					}),
				}),
			),
		);
		expect(screen.getByTestId('alert-query-preview-state')).toHaveTextContent(
			'none',
		);

		fireEvent.click(screen.getByRole('button', { name: 'Logs' }));
		await waitFor(() =>
			expect(onReset).toHaveBeenLastCalledWith(
				expect.objectContaining({
					builder: expect.objectContaining({
						queryData: [
							expect.objectContaining({
								dataSource: 'logs',
								stepInterval: 60,
							}),
						],
					}),
				}),
			),
		);
		expect(screen.getByTestId('alert-query-preview-state')).toHaveTextContent(
			'none',
		);
		expect(onRedirect).toHaveBeenCalledWith(
			expect.objectContaining({
				builder: expect.objectContaining({
					queryData: [
						expect.objectContaining({
							dataSource: 'logs',
							stepInterval: 60,
						}),
					],
				}),
			}),
		);

		fireEvent.click(screen.getByRole('button', { name: 'Run preview' }));
		expect(screen.getByTestId('alert-query-preview-state')).toHaveTextContent(
			'logs:60:1',
		);

		fireEvent.click(screen.getByRole('button', { name: 'Traces' }));
		await waitFor(() =>
			expect(screen.getByTestId('alert-query-preview-state')).toHaveTextContent(
				'none',
			),
		);
	});
});
