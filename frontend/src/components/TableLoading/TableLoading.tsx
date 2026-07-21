import { Skeleton } from 'antd';

import './TableLoading.styles.scss';

function TableLoading(): JSX.Element {
	return (
		<div className="table-loading-state" data-testid="loader">
			<Skeleton.Input
				className="table-loading-state-item"
				size="large"
				block
				active
			/>
			<Skeleton.Input
				className="table-loading-state-item"
				size="large"
				block
				active
			/>
			<Skeleton.Input
				className="table-loading-state-item"
				size="large"
				block
				active
			/>
		</div>
	);
}

export default TableLoading;
