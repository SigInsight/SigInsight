import './OnboardingHeader.styles.scss';

export function OnboardingHeader(): JSX.Element {
	return (
		<div className="header-container">
			<div className="logo-container">
				<img src="/Logos/siginsight-brand-logo.svg" alt="SigInsight" />
				<span className="logo-text">SigInsight</span>
			</div>
		</div>
	);
}
