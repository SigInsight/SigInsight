import { render, screen } from '@testing-library/react';
import { LegendPosition } from 'lib/uPlotV2/components/types';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import type { AlignedData } from 'uplot';

import ChartWrapper from './ChartWrapper';

jest.mock('lib/uPlotV2', () => ({
	UPlotChartHost: ({
		children,
		withContext,
	}: {
		children?: React.ReactNode;
		withContext?: boolean;
	}): JSX.Element => (
		<div data-testid="uplot-host" data-with-context={String(withContext)}>
			{children}
		</div>
	),
}));

jest.mock('react-virtuoso', () => ({
	VirtuosoGrid: ({
		data,
		itemContent,
	}: {
		data: unknown[];
		itemContent: (index: number, item: unknown) => React.ReactNode;
	}): JSX.Element => <>{data.map((item, index) => itemContent(index, item))}</>,
}));

const config = ({
	getLegendItems: jest.fn(() => ({
		1: {
			seriesIndex: 1,
			label: 'request count',
			show: true,
			color: '#fff',
		},
	})),
	addHook: jest.fn(() => jest.fn()),
} as unknown) as UPlotConfigBuilder;

describe('ChartWrapper', () => {
	it('shares one plot context between the plot host and its sibling legend', () => {
		render(
			<ChartWrapper
				config={config}
				data={[[1], [2]] as AlignedData}
				width={600}
				height={400}
				legendConfig={{ position: LegendPosition.BOTTOM }}
				showTooltip={false}
			/>,
		);

		expect(screen.getByText('request count')).toBeInTheDocument();
		expect(screen.getByTestId('uplot-host')).toHaveAttribute(
			'data-with-context',
			'false',
		);
	});
});
