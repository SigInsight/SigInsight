import { fireEvent, render, screen } from '@testing-library/react';

import Controls, { ControlsProps } from './index';

const baseProps: ControlsProps = {
	totalCount: 10,
	countPerPage: 10,
	isLoading: false,
	handleNavigatePrevious: jest.fn(),
	handleNavigateNext: jest.fn(),
	handleCountItemsPerPageChange: jest.fn(),
};

describe('Controls pagination contract', () => {
	it('uses explicit hasNextPage instead of guessing from the row count', () => {
		const handleNavigateNext = jest.fn();
		const { rerender } = render(
			<Controls
				{...baseProps}
				hasNextPage={false}
				handleNavigateNext={handleNavigateNext}
			/>,
		);
		const next = screen.getByRole('button', { name: /next/i });
		expect(next).toBeDisabled();

		rerender(
			<Controls
				{...baseProps}
				hasNextPage
				handleNavigateNext={handleNavigateNext}
			/>,
		);
		expect(next).toBeEnabled();
		fireEvent.click(next);
		expect(handleNavigateNext).toHaveBeenCalledTimes(1);
	});

	it('allows navigating back from an empty page', () => {
		const handleNavigatePrevious = jest.fn();
		render(
			<Controls
				{...baseProps}
				totalCount={0}
				offset={10}
				hasNextPage={false}
				handleNavigatePrevious={handleNavigatePrevious}
			/>,
		);

		const previous = screen.getByRole('button', { name: /previous/i });
		expect(previous).toBeEnabled();
		fireEvent.click(previous);
		expect(handleNavigatePrevious).toHaveBeenCalledTimes(1);
	});
});
