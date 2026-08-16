import { generateGridTitle } from 'lib/generateGridTitle';

describe('generateGridTitle', () => {
	it('extracts text from nested React titles', () => {
		expect(
			generateGridTitle(
				<span>
					<strong>ApDex</strong>
					<span>latency score</span>
				</span>,
			),
		).toBe('ApDex latency score');
	});
});
