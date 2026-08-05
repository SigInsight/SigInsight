import { render, screen } from '@testing-library/react';
import { GetMetricMetadata200 } from 'api/generated/services/sigNoz.schemas';

import Metadata from '../Metadata';
import { MetricMetadata } from '../types';
import { transformMetricMetadata } from '../utils';
import { getMockMetricMetadataData, MOCK_METRIC_NAME } from './testUtlls';

const mockMetricMetadata = transformMetricMetadata(
	getMockMetricMetadataData().data as GetMetricMetadata200,
) as MetricMetadata;

const mockRefetchMetricMetadata = jest.fn();

describe('Metadata', () => {
	it('renders collected metric metadata without editing controls', () => {
		render(
			<Metadata
				metricName={MOCK_METRIC_NAME}
				metadata={mockMetricMetadata}
				isErrorMetricMetadata={false}
				isLoadingMetricMetadata={false}
				refetchMetricMetadata={mockRefetchMetricMetadata}
			/>,
		);

		expect(screen.getByText('Metric Type')).toBeInTheDocument();
		expect(screen.getByText('Gauge')).toBeInTheDocument();
		expect(screen.getByText('Description')).toBeInTheDocument();
		expect(screen.getByText(mockMetricMetadata.description)).toBeInTheDocument();
		expect(screen.getByText('Unit')).toBeInTheDocument();
		expect(screen.getByText(mockMetricMetadata.unit)).toBeInTheDocument();
		expect(screen.getByText('Temporality')).toBeInTheDocument();
		expect(screen.getByText('Delta')).toBeInTheDocument();
		expect(
			screen.queryByRole('button', { name: 'Edit' }),
		).not.toBeInTheDocument();
	});

	it('retains the loading state without exposing an edit action', () => {
		render(
			<Metadata
				metricName={MOCK_METRIC_NAME}
				metadata={null}
				isErrorMetricMetadata={false}
				isLoadingMetricMetadata
				refetchMetricMetadata={mockRefetchMetricMetadata}
			/>,
		);

		expect(screen.getByText('Metadata')).toBeInTheDocument();
		expect(
			screen.queryByRole('button', { name: 'Edit' }),
		).not.toBeInTheDocument();
	});
});
