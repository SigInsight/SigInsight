import EmptyQuickFilterIcon from 'assets/CustomIcons/EmptyQuickFilterIcon';

function LogsQuickFilterEmptyState({
	attributeKey,
}: {
	attributeKey: string;
}): JSX.Element {
	return (
		<section className="go-to-docs">
			<div className="go-to-docs__container">
				<div className="go-to-docs__container-icon">
					<EmptyQuickFilterIcon />
				</div>
				<div className="go-to-docs__container-message">
					{`You'd need to parse out this attribute to start getting them as a fast
            filter.`}
				</div>
			</div>
		</section>
	);
}

export default LogsQuickFilterEmptyState;
