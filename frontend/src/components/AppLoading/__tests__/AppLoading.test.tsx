import { render, screen } from '@testing-library/react';

import AppLoading from '../AppLoading';

describe('AppLoading', () => {
	const SIGNOZ_TEXT = 'SigInsight';
	const TAGLINE_TEXT =
		'OpenTelemetry-Native Logs, Metrics and Traces in a single pane';
	const CONTAINER_SELECTOR = '.app-loading-container';

	it('should render loading screen with dark theme by default', () => {
		render(<AppLoading />);

		// Check if main elements are rendered
		expect(screen.getByAltText(SIGNOZ_TEXT)).toBeInTheDocument();
		expect(screen.getByText(SIGNOZ_TEXT)).toBeInTheDocument();
		expect(screen.getByText(TAGLINE_TEXT)).toBeInTheDocument();

		// Check if dark theme class is applied
		const container = screen.getByText(SIGNOZ_TEXT).closest(CONTAINER_SELECTOR);
		expect(container).toHaveClass('dark');
		expect(container).not.toHaveClass('lightMode');
	});

	it('should have proper structure and content', () => {
		render(<AppLoading />);

		// Check for brand logo
		const logo = screen.getByAltText(SIGNOZ_TEXT);
		expect(logo).toBeInTheDocument();
		expect(logo).toHaveAttribute('src', '/Logos/siginsight-brand-logo.svg');

		// Check for brand title
		const title = screen.getByText(SIGNOZ_TEXT);
		expect(title).toBeInTheDocument();

		// Check for tagline
		const tagline = screen.getByText(TAGLINE_TEXT);
		expect(tagline).toBeInTheDocument();

		// Check for loader
		const loader = document.querySelector('.loader');
		expect(loader).toBeInTheDocument();
	});
});
