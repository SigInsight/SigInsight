import { act, renderHook } from '@testing-library/react';
import history from 'lib/history';
import { UpdateTimeInterval } from 'store/actions';

import { useExplorerTimeRangeSelection } from './useExplorerTimeRangeSelection';

const mockDispatch = jest.fn();
const mockUrlQuery = new URLSearchParams('relativeTime=1h&source=explorer');

jest.mock('react-redux', () => ({
	useDispatch: (): jest.Mock => mockDispatch,
}));
jest.mock('react-router-dom', () => ({
	useLocation: (): { pathname: string } => ({ pathname: '/traces-explorer' }),
}));
jest.mock('hooks/useUrlQuery', () => ({
	__esModule: true,
	default: (): URLSearchParams => mockUrlQuery,
}));
jest.mock('lib/getMinMax', () => ({
	__esModule: true,
	default: (): { minTime: number; maxTime: number } => ({
		minTime: 10000,
		maxTime: 20000,
	}),
}));
jest.mock('lib/history', () => ({
	__esModule: true,
	default: { push: jest.fn() },
}));
jest.mock('store/actions', () => ({
	UpdateTimeInterval: jest.fn((...args: unknown[]) => ({ args })),
}));

describe('useExplorerTimeRangeSelection', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		mockUrlQuery.delete('startTime');
		mockUrlQuery.delete('endTime');
		mockUrlQuery.set('relativeTime', '1h');
		window.history.replaceState({}, '', '/');
	});

	it('updates global time and replaces relative URL time after a drag', () => {
		const { result } = renderHook(() => useExplorerTimeRangeSelection());

		act(() => result.current(10.9, 20.7));

		expect(UpdateTimeInterval).toHaveBeenCalledWith('custom', [10, 20]);
		expect(history.push).toHaveBeenCalledWith(
			'/traces-explorer?source=explorer&startTime=10000&endTime=20000',
		);
		expect(mockUrlQuery.has('relativeTime')).toBe(false);
		expect(mockDispatch).toHaveBeenCalledTimes(1);
	});

	it('restores relative time when browser navigation changes the URL', () => {
		renderHook(() => useExplorerTimeRangeSelection());
		window.history.replaceState({}, '', '/?relativeTime=30m');

		act(() => window.dispatchEvent(new PopStateEvent('popstate')));

		expect(UpdateTimeInterval).toHaveBeenCalledWith('30m');
		expect(mockDispatch).toHaveBeenCalledTimes(1);
	});
});
