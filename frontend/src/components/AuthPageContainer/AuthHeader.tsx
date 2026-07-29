import './AuthHeader.styles.scss';

function AuthHeader(): JSX.Element {
	return (
		<header className="auth-header">
			<div className="auth-header-logo">
				<img
					src="/Logos/siginsight-brand-logo.svg"
					alt="SigInsight"
					className="auth-header-logo-icon"
				/>
				<span className="auth-header-logo-text">SigInsight</span>
			</div>
		</header>
	);
}

export default AuthHeader;
