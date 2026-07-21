import { Typography } from 'antd';
import { MarkdownRenderer } from 'components/MarkdownRenderer/MarkdownRenderer';

import './IntegrationDetailContentTabs.styles.scss';

interface OverviewProps {
	categories: string[];
	overviewContent: string;
}

function Overview(props: OverviewProps): JSX.Element {
	const { categories, overviewContent } = props;

	return (
		<div className="integration-detail-overview">
			<div className="integration-detail-overview-left-container">
				<div className="integration-detail-overview-category">
					<Typography.Text className="heading">Category</Typography.Text>
					<div className="category-tabs">
						{categories.map((category) => (
							<div key={category} className="category-tab">
								{category}
							</div>
						))}
					</div>
				</div>
			</div>
			<div className="integration-detail-overview-right-container">
				<MarkdownRenderer variables={{}} markdownContent={overviewContent} />
			</div>
		</div>
	);
}

export default Overview;
