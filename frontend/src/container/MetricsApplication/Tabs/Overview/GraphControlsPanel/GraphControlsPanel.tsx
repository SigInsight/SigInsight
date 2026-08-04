import { Color } from '@signozhq/design-tokens';
import { Button } from 'antd';
import { DraftingCompass, ScrollText } from 'lucide-react';

import './GraphControlsPanel.styles.scss';

interface GraphControlsPanelProps {
	id: string;
	onViewLogsClick?: () => void;
	onViewTracesClick: () => void;
}

function GraphControlsPanel({
	id,
	onViewLogsClick,
	onViewTracesClick,
}: GraphControlsPanelProps): JSX.Element {
	return (
		<div id={id} className="graph-controls-panel">
			<Button
				type="link"
				icon={<DraftingCompass size={14} />}
				size="small"
				onClick={onViewTracesClick}
				style={{ color: Color.BG_VANILLA_100 }}
			>
				View traces
			</Button>
			{onViewLogsClick && (
				<Button
					type="link"
					icon={<ScrollText size={14} />}
					size="small"
					onClick={onViewLogsClick}
					style={{ color: Color.BG_VANILLA_100 }}
				>
					View logs
				</Button>
			)}
		</div>
	);
}

GraphControlsPanel.defaultProps = {
	onViewLogsClick: undefined,
};

export default GraphControlsPanel;
