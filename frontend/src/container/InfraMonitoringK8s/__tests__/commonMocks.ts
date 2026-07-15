import * as appContextHooks from 'providers/App/App';
import * as timezoneHooks from 'providers/Timezone';

const setupCommonMocks = (): void => {
	const createMockObserver = (): {
		observe: jest.Mock;
		unobserve: jest.Mock;
		disconnect: jest.Mock;
	} => ({
		observe: jest.fn(),
		unobserve: jest.fn(),
		disconnect: jest.fn(),
	});

	global.IntersectionObserver = jest.fn().mockImplementation(createMockObserver);
	global.ResizeObserver = jest.fn().mockImplementation(createMockObserver);

	jest.mock('react-redux', () => ({
		...jest.requireActual('react-redux'),
		useSelector: jest.fn(() => ({
			globalTime: {
				selectedTime: {
					startTime: 1713734400000,
					endTime: 1713738000000,
				},
				maxTime: 1713738000000,
				minTime: 1713734400000,
			},
		})),
	}));

	jest.mock('uplot', () => ({
		paths: {
			spline: jest.fn(),
			bars: jest.fn(),
		},
		default: jest.fn(() => ({
			paths: {
				spline: jest.fn(),
				bars: jest.fn(),
			},
		})),
	}));

	jest.mock('react-router-dom-v5-compat', () => ({
		...jest.requireActual('react-router-dom-v5-compat'),
		useSearchParams: jest.fn().mockReturnValue([
			{
				get: jest.fn(),
				entries: jest.fn(() => []),
				set: jest.fn(),
			},
			jest.fn(),
		]),
		useNavigationType: (): any => 'PUSH',
	}));

	jest.mock('lib/getMinMax', () => ({
		__esModule: true,
		default: jest.fn().mockImplementation(() => ({
			minTime: 1713734400000,
			maxTime: 1713738000000,
		})),
		isValidShortHandDateTimeFormat: jest.fn().mockReturnValue(true),
	}));

	jest.spyOn(appContextHooks, 'useAppContext').mockReturnValue({
		user: {
			role: 'admin',
		},
	} as any);

	jest.spyOn(timezoneHooks, 'useTimezone').mockReturnValue({
		timezone: {
			offset: 0,
		},
		browserTimezone: {
			offset: 0,
		},
	} as any);

	jest.mock('hooks/useSafeNavigate', () => ({
		useSafeNavigate: (): any => ({
			safeNavigate: jest.fn(),
		}),
	}));
};

export default setupCommonMocks;
